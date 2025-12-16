package goadstc

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/symbols"
)

// ReadSymbolValue automatically detects the type of a symbol and returns its parsed value.
// For basic types (INT, REAL, BOOL, etc.), it returns the appropriate Go primitive type.
// For arrays, it returns []interface{} with all elements.
// For structs, it returns map[string]interface{} with all fields.
// This method uses the symbol table to determine the type and reads/parses in one call.
func (c *Client) ReadSymbolValue(ctx context.Context, symbolName string) (interface{}, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, ClassifyError(err, "read_symbol_value")
	}

	c.logger.Debug("reading symbol value with auto-detection", "symbol", symbolName)

	// Parse array notation if present (e.g., "MAIN.array[5]")
	baseName, arrayIndices, err := parseArrayAccess(symbolName)
	if err != nil {
		return nil, ClassifyError(err, "read_symbol_value")
	}

	// Get symbol information
	symbol, err := c.symbolTable.Get(baseName)
	if err != nil {
		return nil, ClassifyError(err, "read_symbol_value")
	}

	// Read the raw data
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return nil, ClassifyError(err, "read_symbol_value")
	}

	// Auto-detect and parse based on type
	value, err := c.parseSymbolValue(ctx, data, symbol, len(arrayIndices) > 0)
	if err != nil {
		return nil, ClassifyError(err, "read_symbol_value")
	}

	c.logger.Debug("successfully read symbol value", "symbol", symbolName, "type", symbol.Type.Name)
	return value, nil
}

// ReadMultipleSymbolValues reads multiple symbols in individual requests and returns a map of results.
// TODO: Future enhancement - use SumCommand (0xF080) for batched reads in single request.
func (c *Client) ReadMultipleSymbolValues(ctx context.Context, symbolNames ...string) (map[string]interface{}, error) {
	if len(symbolNames) == 0 {
		return make(map[string]interface{}), nil
	}

	c.logger.Debug("reading multiple symbol values", "count", len(symbolNames))

	results := make(map[string]interface{}, len(symbolNames))
	for _, name := range symbolNames {
		value, err := c.ReadSymbolValue(ctx, name)
		if err != nil {
			// Include error in results instead of failing completely
			results[name] = fmt.Errorf("read failed: %w", err)
			c.logger.Warn("failed to read symbol", "symbol", name, "error", err)
		} else {
			results[name] = value
		}
	}

	return results, nil
}

// parseSymbolValue determines the appropriate parser based on symbol type information.
func (c *Client) parseSymbolValue(ctx context.Context, data []byte, symbol *symbols.Symbol, isArrayElement bool) (interface{}, error) {
	typeInfo := symbol.Type

	// Handle array access - element should be parsed as its base type
	if isArrayElement && typeInfo.IsArray {
		// Extract element type from array notation (e.g., "ARRAY [0..9] OF INT" -> "INT")
		if elementTypeName, isArray := extractArrayElementType(typeInfo.Name); isArray {
			// Parse as simple type or struct depending on element type
			if isSimpleTypeName(elementTypeName) {
				return c.parseSimpleTypeByName(data, elementTypeName)
			}
			// Element is a struct - fetch its type info and parse
			return c.parseStructWithTypeInfo(ctx, data, elementTypeName)
		}
	}

	// Handle full arrays (not array element access)
	if typeInfo.IsArray && !isArrayElement {
		return c.parseArrayValue(ctx, data, typeInfo)
	}

	// Handle structs
	if typeInfo.IsStruct || strings.Contains(typeInfo.Name, "STRUCT") {
		return c.parseStructWithTypeInfo(ctx, data, typeInfo.Name)
	}

	// Handle simple types by data type ID
	if value := parseSimpleTypeByID(data, typeInfo.BaseType); value != nil {
		return value, nil
	}

	// Try parsing by type name as fallback
	if value, err := c.parseSimpleTypeByName(data, typeInfo.Name); err == nil {
		return value, nil
	}

	// Unknown type - return raw data
	c.logger.Warn("unknown type, returning raw data", "type", typeInfo.Name, "baseType", typeInfo.BaseType)
	return data, nil
}

// parseArrayValue parses an array into []interface{}.
func (c *Client) parseArrayValue(ctx context.Context, data []byte, typeInfo symbols.TypeInfo) (interface{}, error) {
	// Calculate total elements
	totalElements := uint32(1)
	for _, dim := range typeInfo.ArrayDims {
		totalElements *= dim
	}

	if totalElements == 0 {
		return []interface{}{}, nil
	}

	// Extract element type
	elementTypeName, _ := extractArrayElementType(typeInfo.Name)
	elementSize := typeInfo.Size / totalElements

	if elementSize == 0 {
		return nil, fmt.Errorf("invalid element size: total=%d, elements=%d", typeInfo.Size, totalElements)
	}

	result := make([]interface{}, 0, totalElements)

	// Parse each element
	for i := uint32(0); i < totalElements; i++ {
		offset := i * elementSize
		if offset+elementSize > uint32(len(data)) {
			break
		}

		elementData := data[offset : offset+elementSize]

		var element interface{}
		var err error

		if isSimpleTypeName(elementTypeName) {
			element, err = c.parseSimpleTypeByName(elementData, elementTypeName)
		} else {
			// Complex type (struct)
			element, err = c.parseStructWithTypeInfo(ctx, elementData, elementTypeName)
		}

		if err != nil {
			c.logger.Warn("failed to parse array element", "index", i, "error", err)
			element = elementData // Fallback to raw data
		}

		result = append(result, element)
	}

	return result, nil
}

// parseStructWithTypeInfo parses a struct by fetching or using cached type information.
func (c *Client) parseStructWithTypeInfo(ctx context.Context, data []byte, typeName string) (interface{}, error) {
	// Get or fetch type info
	typeInfo, err := c.getOrFetchTypeInfo(ctx, typeName)
	if err != nil {
		c.logger.Warn("could not get type info for struct", "type", typeName, "error", err)
		// Return raw data with metadata
		return map[string]interface{}{
			"_raw":  data,
			"_type": typeName,
			"_size": len(data),
			"_note": "Type information not available",
		}, nil
	}

	if len(typeInfo.Fields) == 0 {
		return map[string]interface{}{
			"_raw":  data,
			"_type": typeName,
			"_size": len(data),
		}, nil
	}

	// Parse all fields
	result := make(map[string]interface{}, len(typeInfo.Fields))
	for _, field := range typeInfo.Fields {
		if int(field.Offset)+int(field.Type.Size) > len(data) {
			continue
		}
		fieldData := data[field.Offset : field.Offset+field.Type.Size]
		result[field.Name] = parseFieldValue(fieldData, field.Type)
	}

	return result, nil
}

// parseSimpleTypeByName parses a value based on TwinCAT type name string.
func (c *Client) parseSimpleTypeByName(data []byte, typeName string) (interface{}, error) {
	typeName = strings.ToUpper(strings.TrimSpace(typeName))

	switch typeName {
	case "BOOL":
		if len(data) < 1 {
			return nil, fmt.Errorf("insufficient data for BOOL")
		}
		return data[0] != 0, nil

	case "SINT", "INT8":
		if len(data) < 1 {
			return nil, fmt.Errorf("insufficient data for INT8")
		}
		return int8(data[0]), nil

	case "USINT", "BYTE", "UINT8":
		if len(data) < 1 {
			return nil, fmt.Errorf("insufficient data for UINT8")
		}
		return uint8(data[0]), nil

	case "INT", "INT16":
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for INT16")
		}
		return int16(binary.LittleEndian.Uint16(data)), nil

	case "UINT", "WORD", "UINT16":
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for UINT16")
		}
		return binary.LittleEndian.Uint16(data), nil

	case "DINT", "INT32":
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for INT32")
		}
		return int32(binary.LittleEndian.Uint32(data)), nil

	case "UDINT", "DWORD", "UINT32":
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for UINT32")
		}
		return binary.LittleEndian.Uint32(data), nil

	case "LINT", "INT64":
		if len(data) < 8 {
			return nil, fmt.Errorf("insufficient data for INT64")
		}
		return int64(binary.LittleEndian.Uint64(data)), nil

	case "ULINT", "LWORD", "UINT64":
		if len(data) < 8 {
			return nil, fmt.Errorf("insufficient data for UINT64")
		}
		return binary.LittleEndian.Uint64(data), nil

	case "REAL", "FLOAT":
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for REAL")
		}
		bits := binary.LittleEndian.Uint32(data)
		return math.Float32frombits(bits), nil

	case "LREAL", "DOUBLE":
		if len(data) < 8 {
			return nil, fmt.Errorf("insufficient data for LREAL")
		}
		bits := binary.LittleEndian.Uint64(data)
		return math.Float64frombits(bits), nil

	case "STRING":
		for i, b := range data {
			if b == 0 {
				return string(data[:i]), nil
			}
		}
		return string(data), nil

	case "TIME":
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for TIME")
		}
		ms := int32(binary.LittleEndian.Uint32(data))
		return time.Duration(ms) * time.Millisecond, nil

	case "DATE", "DATE_AND_TIME", "DT":
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for DATE")
		}
		seconds := binary.LittleEndian.Uint32(data)
		return time.Unix(int64(seconds), 0), nil

	case "TIME_OF_DAY", "TOD":
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for TIME_OF_DAY")
		}
		ms := binary.LittleEndian.Uint32(data)
		return time.Duration(ms) * time.Millisecond, nil

	default:
		return nil, fmt.Errorf("unknown type: %s", typeName)
	}
}

// parseSimpleTypeByID parses a value based on symbols.DataType ID.
func parseSimpleTypeByID(data []byte, dataType symbols.DataType) interface{} {
	switch dataType {
	case symbols.DataTypeBool:
		if len(data) >= 1 {
			return data[0] != 0
		}
	case symbols.DataTypeInt8:
		if len(data) >= 1 {
			return int8(data[0])
		}
	case symbols.DataTypeUInt8:
		if len(data) >= 1 {
			return uint8(data[0])
		}
	case symbols.DataTypeInt16:
		if len(data) >= 2 {
			return int16(binary.LittleEndian.Uint16(data))
		}
	case symbols.DataTypeUInt16:
		if len(data) >= 2 {
			return binary.LittleEndian.Uint16(data)
		}
	case symbols.DataTypeInt32:
		if len(data) >= 4 {
			return int32(binary.LittleEndian.Uint32(data))
		}
	case symbols.DataTypeUInt32:
		if len(data) >= 4 {
			return binary.LittleEndian.Uint32(data)
		}
	case symbols.DataTypeInt64:
		if len(data) >= 8 {
			return int64(binary.LittleEndian.Uint64(data))
		}
	case symbols.DataTypeUInt64:
		if len(data) >= 8 {
			return binary.LittleEndian.Uint64(data)
		}
	case symbols.DataTypeReal32:
		if len(data) >= 4 {
			bits := binary.LittleEndian.Uint32(data)
			return math.Float32frombits(bits)
		}
	case symbols.DataTypeReal64:
		if len(data) >= 8 {
			bits := binary.LittleEndian.Uint64(data)
			return math.Float64frombits(bits)
		}
	case symbols.DataTypeString:
		for i, b := range data {
			if b == 0 {
				return string(data[:i])
			}
		}
		return string(data)
	}
	return nil
}

// isSimpleTypeName checks if a type name represents a simple (non-struct) type.
func isSimpleTypeName(typeName string) bool {
	typeName = strings.ToUpper(strings.TrimSpace(typeName))
	simpleTypes := []string{
		"BOOL", "SINT", "INT", "DINT", "LINT",
		"USINT", "UINT", "UDINT", "ULINT",
		"BYTE", "WORD", "DWORD", "LWORD",
		"INT8", "INT16", "INT32", "INT64",
		"UINT8", "UINT16", "UINT32", "UINT64",
		"REAL", "LREAL", "FLOAT", "DOUBLE",
		"STRING", "WSTRING",
		"TIME", "DATE", "TIME_OF_DAY", "TOD", "DATE_AND_TIME", "DT",
	}

	for _, st := range simpleTypes {
		if typeName == st {
			return true
		}
	}
	return false
}

// WriteSymbolValue automatically encodes and writes a value to a symbol based on its type.
// Supports basic types (int, float, bool, string), time.Duration, and time.Time.
// For complex types (structs, arrays), use the specific Write methods or WriteSymbol with raw bytes.
func (c *Client) WriteSymbolValue(ctx context.Context, symbolName string, value interface{}) error {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return ClassifyError(err, "write_symbol_value")
	}

	c.logger.Debug("writing symbol value with auto-encoding", "symbol", symbolName)

	// Parse array notation if present
	baseName, _, err := parseArrayAccess(symbolName)
	if err != nil {
		return ClassifyError(err, "write_symbol_value")
	}

	// Get symbol information
	symbol, err := c.symbolTable.Get(baseName)
	if err != nil {
		return ClassifyError(err, "write_symbol_value")
	}

	// Encode value based on Go type
	data, err := c.encodeSymbolValue(value, symbol)
	if err != nil {
		return ClassifyError(err, "write_symbol_value")
	}

	// Write the encoded data
	if err := c.WriteSymbol(ctx, symbolName, data); err != nil {
		return ClassifyError(err, "write_symbol_value")
	}

	c.logger.Debug("successfully wrote symbol value", "symbol", symbolName, "type", symbol.Type.Name)
	return nil
}

// encodeSymbolValue encodes a Go value into bytes based on the symbol's type.
func (c *Client) encodeSymbolValue(value interface{}, symbol *symbols.Symbol) ([]byte, error) {
	// Handle nil
	if value == nil {
		return make([]byte, symbol.Size), nil
	}

	// Try type-specific encoding based on Go type
	switch v := value.(type) {
	case bool:
		data := make([]byte, 1)
		if v {
			data[0] = 1
		}
		return data, nil

	case int8:
		return []byte{byte(v)}, nil

	case uint8:
		return []byte{v}, nil

	case int16:
		data := make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(v))
		return data, nil

	case uint16:
		data := make([]byte, 2)
		binary.LittleEndian.PutUint16(data, v)
		return data, nil

	case int32:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(v))
		return data, nil

	case uint32:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, v)
		return data, nil

	case int64:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(v))
		return data, nil

	case uint64:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, v)
		return data, nil

	case int:
		// Default int handling - use symbol size to determine encoding
		if symbol.Size == 2 {
			data := make([]byte, 2)
			binary.LittleEndian.PutUint16(data, uint16(v))
			return data, nil
		} else if symbol.Size == 4 {
			data := make([]byte, 4)
			binary.LittleEndian.PutUint32(data, uint32(v))
			return data, nil
		} else if symbol.Size == 8 {
			data := make([]byte, 8)
			binary.LittleEndian.PutUint64(data, uint64(v))
			return data, nil
		}
		return nil, fmt.Errorf("cannot encode int to size %d", symbol.Size)

	case float32:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, math.Float32bits(v))
		return data, nil

	case float64:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, math.Float64bits(v))
		return data, nil

	case string:
		// String encoding - null-terminated, padded to symbol size
		data := make([]byte, symbol.Size)
		maxLen := int(symbol.Size) - 1
		if len(v) > maxLen {
			v = v[:maxLen]
		}
		copy(data, []byte(v))
		return data, nil

	case time.Duration:
		// TIME or TIME_OF_DAY - milliseconds as 32-bit int/uint
		ms := uint32(v / time.Millisecond)
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, ms)
		return data, nil

	case time.Time:
		// DATE, DATE_AND_TIME - seconds since epoch as 32-bit uint
		secs := uint32(v.Unix())
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, secs)
		return data, nil

	case []byte:
		// Raw bytes - pad/truncate to symbol size
		data := make([]byte, symbol.Size)
		copy(data, v)
		return data, nil

	default:
		return nil, fmt.Errorf("unsupported type for auto-encoding: %T (use type-specific Write method or WriteSymbol)", value)
	}
}

// WriteStructFields writes multiple fields to a struct by reading the entire struct,
// modifying the specified fields at their byte offsets, and writing the struct back.
// This is useful when individual field symbols aren't exported in the PLC.
//
// fieldValues is a map where keys are field names and values are the Go values to write.
// The function automatically encodes each value based on its type information from the PLC.
//
// Example:
//
//	err := client.WriteStructFields(ctx, "MAIN.myStruct", map[string]interface{}{
//	    "temperature": float32(25.5),
//	    "enabled": true,
//	    "counter": int16(42),
//	})
func (c *Client) WriteStructFields(ctx context.Context, symbolName string, fieldValues map[string]interface{}) error {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return ClassifyError(err, "write_struct_fields")
	}

	c.logger.Debug("writing struct fields", "symbol", symbolName, "fieldCount", len(fieldValues))

	// Get symbol information
	symbol, err := c.symbolTable.Get(symbolName)
	if err != nil {
		return ClassifyError(fmt.Errorf("symbol '%s' not found: %w", symbolName, err), "write_struct_fields")
	}

	// Ensure it's a struct type
	if symbol.Type.Name == "" {
		return ClassifyError(fmt.Errorf("symbol '%s' is not a struct type", symbolName), "write_struct_fields")
	}

	// Get type information for the struct
	typeInfo, err := c.getOrFetchTypeInfo(ctx, symbol.Type.Name)
	if err != nil {
		return ClassifyError(fmt.Errorf("failed to get type info for '%s': %w", symbol.Type.Name, err), "write_struct_fields")
	}

	if len(typeInfo.Fields) == 0 {
		return ClassifyError(fmt.Errorf("struct '%s' has no fields", symbolName), "write_struct_fields")
	}

	// Read the current struct data
	structData, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return ClassifyError(fmt.Errorf("failed to read struct '%s': %w", symbolName, err), "write_struct_fields")
	}

	// Modify fields at their offsets
	for fieldName, fieldValue := range fieldValues {
		// Find field in type info
		var fieldInfo *symbols.FieldInfo
		for i := range typeInfo.Fields {
			if typeInfo.Fields[i].Name == fieldName {
				fieldInfo = &typeInfo.Fields[i]
				break
			}
		}

		if fieldInfo == nil {
			return ClassifyError(fmt.Errorf("field '%s' not found in struct '%s'", fieldName, symbolName), "write_struct_fields")
		}

		// Encode the field value
		fieldSymbol := &symbols.Symbol{
			Type: fieldInfo.Type,
			Size: fieldInfo.Type.Size,
		}

		encodedField, err := c.encodeSymbolValue(fieldValue, fieldSymbol)
		if err != nil {
			return ClassifyError(fmt.Errorf("failed to encode field '%s': %w", fieldName, err), "write_struct_fields")
		}

		// Check bounds
		if fieldInfo.Offset+fieldInfo.Type.Size > uint32(len(structData)) {
			return ClassifyError(fmt.Errorf("field '%s' offset %d + size %d exceeds struct size %d",
				fieldName, fieldInfo.Offset, fieldInfo.Type.Size, len(structData)), "write_struct_fields")
		}

		// Write encoded field at offset
		copy(structData[fieldInfo.Offset:fieldInfo.Offset+fieldInfo.Type.Size], encodedField)
		c.logger.Debug("modified field in struct data", "field", fieldName, "offset", fieldInfo.Offset, "size", fieldInfo.Type.Size)
	}

	// Write the modified struct back
	if err := c.WriteSymbol(ctx, symbolName, structData); err != nil {
		return ClassifyError(fmt.Errorf("failed to write modified struct: %w", err), "write_struct_fields")
	}

	c.logger.Debug("successfully wrote struct fields", "symbol", symbolName, "fieldCount", len(fieldValues))
	return nil
}
