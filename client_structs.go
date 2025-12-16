package goadstc

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/symbols"
)

// Struct field access methods (Milestone 4)

// ReadStructField reads a field from a struct by path (e.g., "MAIN.myStruct.field1").
// This is a simplified implementation that reads the entire struct and extracts the field.
// For complex structs with detailed type information, use the type-safe methods directly.
func (c *Client) ReadStructField(ctx context.Context, structPath string, fieldName string) ([]byte, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, err
	}

	// Get the struct symbol
	symbol, err := c.symbolTable.Get(structPath)
	if err != nil {
		return nil, fmt.Errorf("get symbol %q: %w", structPath, err)
	}

	if !symbol.Type.IsStruct {
		return nil, fmt.Errorf("%q is not a struct type", structPath)
	}

	// For now, read the entire struct
	// In a full implementation, we would parse the type information to find field offset
	structData, err := c.ReadSymbol(ctx, structPath)
	if err != nil {
		return nil, fmt.Errorf("read struct %q: %w", structPath, err)
	}

	// This is a simplified version - a full implementation would need
	// the PLC to provide detailed type information including field offsets
	return structData, fmt.Errorf("field extraction requires detailed type information from PLC")
}

// WriteStructField writes a field to a struct by path.
// This is a simplified implementation that requires reading the entire struct,
// modifying the field, and writing back.
func (c *Client) WriteStructField(ctx context.Context, structPath string, fieldName string, fieldData []byte) error {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return err
	}

	// Get the struct symbol
	symbol, err := c.symbolTable.Get(structPath)
	if err != nil {
		return fmt.Errorf("get symbol %q: %w", structPath, err)
	}

	if !symbol.Type.IsStruct {
		return fmt.Errorf("%q is not a struct type", structPath)
	}

	// For now, return an error indicating limitation
	// A full implementation would need detailed type information from the PLC
	return fmt.Errorf("struct field writing requires detailed type information from PLC")
}

// ReadStructFieldInt16 reads an INT16 field from a struct.
// This uses direct symbol path like "MAIN.myStruct.field1".
func (c *Client) ReadStructFieldInt16(ctx context.Context, fieldPath string) (int16, error) {
	return c.ReadInt16(ctx, fieldPath)
}

// ReadStructFieldUint16 reads a UINT16 field from a struct.
func (c *Client) ReadStructFieldUint16(ctx context.Context, fieldPath string) (uint16, error) {
	return c.ReadUint16(ctx, fieldPath)
}

// ReadStructFieldInt32 reads an INT32 field from a struct.
func (c *Client) ReadStructFieldInt32(ctx context.Context, fieldPath string) (int32, error) {
	return c.ReadInt32(ctx, fieldPath)
}

// ReadStructFieldUint32 reads a UINT32 field from a struct.
func (c *Client) ReadStructFieldUint32(ctx context.Context, fieldPath string) (uint32, error) {
	return c.ReadUint32(ctx, fieldPath)
}

// ReadStructFieldBool reads a BOOL field from a struct.
func (c *Client) ReadStructFieldBool(ctx context.Context, fieldPath string) (bool, error) {
	return c.ReadBool(ctx, fieldPath)
}

// WriteStructFieldInt16 writes an INT16 field to a struct.
func (c *Client) WriteStructFieldInt16(ctx context.Context, fieldPath string, value int16) error {
	return c.WriteInt16(ctx, fieldPath, value)
}

// WriteStructFieldUint16 writes a UINT16 field to a struct.
func (c *Client) WriteStructFieldUint16(ctx context.Context, fieldPath string, value uint16) error {
	return c.WriteUint16(ctx, fieldPath, value)
}

// WriteStructFieldInt32 writes an INT32 field to a struct.
func (c *Client) WriteStructFieldInt32(ctx context.Context, fieldPath string, value int32) error {
	return c.WriteInt32(ctx, fieldPath, value)
}

// WriteStructFieldUint32 writes a UINT32 field to a struct.
func (c *Client) WriteStructFieldUint32(ctx context.Context, fieldPath string, value uint32) error {
	return c.WriteUint32(ctx, fieldPath, value)
}

// WriteStructFieldBool writes a BOOL field to a struct.
func (c *Client) WriteStructFieldBool(ctx context.Context, fieldPath string, value bool) error {
	return c.WriteBool(ctx, fieldPath, value)
}

// GetDataTypeUploadInfo retrieves information about the data type table.
func (c *Client) GetDataTypeUploadInfo(ctx context.Context) (dataTypeCount, dataTypeSize uint32, err error) {
	// Use Read command with ADSIGRP_SYM_DT_UPLOADINFO (0xF010)
	readData, err := c.Read(ctx, 0xF010, 0, 0x30) // 48 bytes for upload info
	if err != nil {
		return 0, 0, fmt.Errorf("get data type upload info: %w", err)
	}

	var resp ads.DataTypeUploadInfoResponse
	if err := resp.UnmarshalBinary(readData); err != nil {
		return 0, 0, fmt.Errorf("unmarshal data type upload info: %w", err)
	}

	return resp.DataTypeCount, resp.DataTypeSize, nil
}

// UploadDataTypeTable retrieves the complete data type table from the PLC.
func (c *Client) UploadDataTypeTable(ctx context.Context) ([]byte, error) {
	// Get data type info first
	_, dataTypeSize, err := c.GetDataTypeUploadInfo(ctx)
	if err != nil {
		return nil, err
	}

	if dataTypeSize == 0 {
		return nil, fmt.Errorf("data type size is 0")
	}

	// Use Read command with ADSIGRP_SYM_DT_UPLOAD (0xF011)
	readLength := dataTypeSize + 1024 // Add buffer
	readData, err := c.Read(ctx, 0xF011, 0, readLength)
	if err != nil {
		return nil, fmt.Errorf("upload data type table: %w", err)
	}

	return readData, nil
}

// RegisterType registers a custom type definition for automatic struct parsing.
// This allows ReadStructAsMap to automatically parse structs based on registered type information.
func (c *Client) RegisterType(typeInfo symbols.TypeInfo) {
	c.typeRegistryMu.Lock()
	defer c.typeRegistryMu.Unlock()
	c.typeRegistry.Register(typeInfo.Name, typeInfo)
}

// GetRegisteredType retrieves a registered type definition.
func (c *Client) GetRegisteredType(typeName string) (symbols.TypeInfo, bool) {
	c.typeRegistryMu.RLock()
	defer c.typeRegistryMu.RUnlock()
	return c.typeRegistry.Get(typeName)
}

// ListRegisteredTypes returns all registered type names.
func (c *Client) ListRegisteredTypes() []string {
	c.typeRegistryMu.RLock()
	defer c.typeRegistryMu.RUnlock()
	return c.typeRegistry.List()
}

// getOrFetchTypeInfo gets type info from registry or fetches from PLC if not cached.
func (c *Client) getOrFetchTypeInfo(ctx context.Context, typeName string) (symbols.TypeInfo, error) {
	// Try to get from registry first
	c.typeRegistryMu.RLock()
	if typeInfo, exists := c.typeRegistry.Get(typeName); exists {
		c.typeRegistryMu.RUnlock()
		return typeInfo, nil
	}
	c.typeRegistryMu.RUnlock()

	// Not in registry, fetch from PLC
	typeInfo, err := c.fetchTypeInfoFromPLC(ctx, typeName)
	if err != nil {
		return symbols.TypeInfo{}, err
	}

	// Cache it
	c.typeRegistryMu.Lock()
	c.typeRegistry.Register(typeName, typeInfo)
	c.typeRegistryMu.Unlock()

	return typeInfo, nil
}

// fetchTypeInfoFromPLC retrieves type information from the PLC using ADSIGRP_SYM_DT_UPLOAD (0xF011).
func (c *Client) fetchTypeInfoFromPLC(ctx context.Context, typeName string) (symbols.TypeInfo, error) {
	// Use ReadWrite command with ADSIGRP_SYM_DT_UPLOAD (0xF011)
	typeNameBytes := []byte(typeName)
	typeNameBytes = append(typeNameBytes, 0) // Null terminator

	readData, err := c.ReadWrite(ctx, 0xF011, 0, 0xFFFF, typeNameBytes)
	if err != nil {
		return symbols.TypeInfo{}, fmt.Errorf("read type info from PLC: %w", err)
	}

	if len(readData) < 42 {
		return symbols.TypeInfo{}, fmt.Errorf("response too short for type info: %d bytes", len(readData))
	}

	// Parse data type entry structure according to ADS specification:
	// Offset 0: entryLength (4 bytes)
	// Offset 16: size (4 bytes)
	// Offset 20: offs (4 bytes)
	// Offset 24: dataType (4 bytes)
	// Offset 32: nameLength (2 bytes)
	// Offset 34: typeLength (2 bytes)
	// Offset 36: commentLength (2 bytes)
	// Offset 40: subItems (2 bytes) <- number of fields

	typeSize := binary.LittleEndian.Uint32(readData[16:20])
	dataTypeID := binary.LittleEndian.Uint32(readData[24:28])
	nameLength := binary.LittleEndian.Uint16(readData[32:34])
	typeLength := binary.LittleEndian.Uint16(readData[34:36])
	commentLength := binary.LittleEndian.Uint16(readData[36:38])
	subItems := binary.LittleEndian.Uint16(readData[40:42])

	typeInfo := symbols.TypeInfo{
		Name:     typeName,
		BaseType: symbols.DataType(dataTypeID),
		Size:     typeSize,
		IsStruct: subItems > 0,
		Fields:   make([]symbols.FieldInfo, 0, subItems),
	}

	if subItems == 0 {
		return typeInfo, nil // No fields (primitive type)
	}

	// Calculate offset to sub-items
	offset := 42 + int(nameLength) + 1 + int(typeLength) + 1 + int(commentLength) + 1

	// Parse each sub-item (field)
	for i := 0; i < int(subItems) && offset+42 <= len(readData); i++ {
		fieldSize := binary.LittleEndian.Uint32(readData[offset+16 : offset+20])
		fieldOffset := binary.LittleEndian.Uint32(readData[offset+20 : offset+24])
		fieldDataType := binary.LittleEndian.Uint32(readData[offset+24 : offset+28])
		fieldNameLen := binary.LittleEndian.Uint16(readData[offset+32 : offset+34])
		fieldTypeLen := binary.LittleEndian.Uint16(readData[offset+34 : offset+36])

		// Extract field name
		fieldNameStart := offset + 42
		fieldNameEnd := fieldNameStart + int(fieldNameLen)
		if fieldNameEnd > len(readData) {
			break
		}
		fieldName := string(readData[fieldNameStart:fieldNameEnd])
		// Remove null terminator
		for idx := 0; idx < len(fieldName); idx++ {
			if fieldName[idx] == 0 {
				fieldName = fieldName[:idx]
				break
			}
		}

		// Extract field type name
		fieldTypeStart := fieldNameEnd + 1
		fieldTypeEnd := fieldTypeStart + int(fieldTypeLen)
		if fieldTypeEnd > len(readData) {
			break
		}
		fieldTypeName := string(readData[fieldTypeStart:fieldTypeEnd])
		// Remove null terminator
		for idx := 0; idx < len(fieldTypeName); idx++ {
			if fieldTypeName[idx] == 0 {
				fieldTypeName = fieldTypeName[:idx]
				break
			}
		}

		// Create field info
		fieldInfo := symbols.FieldInfo{
			Name:   fieldName,
			Offset: fieldOffset,
			Type: symbols.TypeInfo{
				Name:     fieldTypeName,
				BaseType: symbols.DataType(fieldDataType),
				Size:     fieldSize,
			},
		}

		// Check if field is a nested struct - recursively fetch its type info
		if fieldDataType == 65 || !isSimpleDataType(symbols.DataType(fieldDataType)) {
			fieldInfo.Type.IsStruct = true
			// Recursively fetch nested struct type info
			if nestedTypeInfo, err := c.fetchTypeInfoFromPLC(ctx, fieldTypeName); err == nil {
				fieldInfo.Type = nestedTypeInfo
			}
		}

		typeInfo.Fields = append(typeInfo.Fields, fieldInfo)

		// Move to next sub-item
		entryLength := binary.LittleEndian.Uint32(readData[offset : offset+4])
		offset += int(entryLength)
	}

	return typeInfo, nil
}

// isSimpleDataType checks if a data type is a simple (non-struct) type.
func isSimpleDataType(dt symbols.DataType) bool {
	switch dt {
	case symbols.DataTypeInt8, symbols.DataTypeUInt8,
		symbols.DataTypeInt16, symbols.DataTypeUInt16,
		symbols.DataTypeInt32, symbols.DataTypeUInt32,
		symbols.DataTypeInt64, symbols.DataTypeUInt64,
		symbols.DataTypeReal32, symbols.DataTypeReal64,
		symbols.DataTypeBool, symbols.DataTypeString:
		return true
	default:
		return false
	}
}

// ReadStructAsMap reads a struct symbol and returns its fields as a map.
// The map keys are field names and values are interface{} containing the parsed values.
// If the struct type is registered via RegisterType, it will use that information for parsing.
func (c *Client) ReadStructAsMap(ctx context.Context, symbolName string) (map[string]interface{}, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, err
	}

	// Get symbol and validate type
	symbol, structTypeName, err := c.getAndValidateStructSymbol(symbolName)
	if err != nil {
		return nil, err
	}

	// Read the struct data
	structData, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return nil, fmt.Errorf("read struct %q: %w", symbolName, err)
	}

	// Get or fetch type information
	typeInfo, hasTypeInfo := c.resolveTypeInfo(ctx, structTypeName)

	// Parse struct using available type information
	return c.parseStructData(structData, typeInfo, hasTypeInfo, symbol)
}

// getAndValidateStructSymbol gets the symbol and validates it's a struct type.
func (c *Client) getAndValidateStructSymbol(symbolName string) (*symbols.Symbol, string, error) {
	// Parse array notation if present
	baseName, _, err := parseArrayAccess(symbolName)
	if err != nil {
		return nil, "", err
	}

	// Get the base symbol to check type
	symbol, err := c.symbolTable.Get(baseName)
	if err != nil {
		return nil, "", fmt.Errorf("get symbol %q: %w", baseName, err)
	}

	// Determine the struct type name (handle array of structs)
	structTypeName := symbol.Type.Name
	if elementType, isArray := extractArrayElementType(symbol.Type.Name); isArray {
		structTypeName = elementType
	}

	// Verify it's a struct type
	if !symbol.Type.IsStruct && !strings.Contains(symbol.Type.Name, "ARRAY") {
		return nil, "", fmt.Errorf("%q is not a struct type", symbolName)
	}

	return symbol, structTypeName, nil
}

// resolveTypeInfo gets type info from registry or fetches from PLC.
func (c *Client) resolveTypeInfo(ctx context.Context, structTypeName string) (symbols.TypeInfo, bool) {
	// Check if type is registered in the type registry
	c.typeRegistryMu.RLock()
	typeInfo, hasTypeInfo := c.typeRegistry.Get(structTypeName)
	c.typeRegistryMu.RUnlock()

	// If not registered or no fields, try to fetch from PLC
	if !hasTypeInfo || len(typeInfo.Fields) == 0 {
		if fetchedTypeInfo, err := c.fetchTypeInfoFromPLC(ctx, structTypeName); err == nil {
			typeInfo = fetchedTypeInfo
			hasTypeInfo = true
			// Cache it for future use
			c.typeRegistryMu.Lock()
			c.typeRegistry.Register(structTypeName, typeInfo)
			c.typeRegistryMu.Unlock()
		}
	}

	return typeInfo, hasTypeInfo
}

// parseStructData parses struct data using available type information.
func (c *Client) parseStructData(structData []byte, typeInfo symbols.TypeInfo, hasTypeInfo bool, symbol *symbols.Symbol) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Use type info if available
	if hasTypeInfo && len(typeInfo.Fields) > 0 {
		parseFieldsFromTypeInfo(result, structData, typeInfo.Fields)
		return result, nil
	}

	// Fall back to symbol table field information
	if len(symbol.Type.Fields) > 0 {
		parseFieldsFromTypeInfo(result, structData, symbol.Type.Fields)
		return result, nil
	}

	// No detailed field info available
	addRawStructInfo(result, structData, symbol.Type.Name)
	return result, nil
}

// parseFieldsFromTypeInfo parses fields using type information.
func parseFieldsFromTypeInfo(result map[string]interface{}, structData []byte, fields []symbols.FieldInfo) {
	for _, field := range fields {
		if int(field.Offset)+int(field.Type.Size) > len(structData) {
			continue // Skip fields beyond data bounds
		}
		fieldData := structData[field.Offset : field.Offset+field.Type.Size]
		result[field.Name] = parseFieldValue(fieldData, field.Type)
	}
}

// addRawStructInfo adds raw struct information when type info is not available.
func addRawStructInfo(result map[string]interface{}, structData []byte, typeName string) {
	result["_raw"] = structData
	result["_size"] = len(structData)
	result["_type"] = typeName
	result["_note"] = "Type information not available from PLC. Data type upload may not be supported by this TwinCAT version."
}

// parseFieldValue parses a field value based on its type.
func parseFieldValue(data []byte, typeInfo symbols.TypeInfo) interface{} {
	if len(data) == 0 {
		return nil
	}

	// Handle arrays
	if typeInfo.IsArray {
		return fmt.Sprintf("<array %d bytes>", len(data))
	}

	// Handle nested structs
	if typeInfo.IsStruct {
		return parseNestedStruct(data, typeInfo)
	}

	// Parse simple types
	if value := parseSimpleType(data, typeInfo.BaseType); value != nil {
		return value
	}

	// Default: return hex string
	return fmt.Sprintf("0x%x", data)
}

// parseNestedStruct handles parsing of nested struct types.
func parseNestedStruct(data []byte, typeInfo symbols.TypeInfo) interface{} {
	if len(typeInfo.Fields) == 0 {
		return fmt.Sprintf("<struct %s, %d bytes>", typeInfo.Name, len(data))
	}

	nestedResult := make(map[string]interface{})
	for _, field := range typeInfo.Fields {
		if int(field.Offset)+int(field.Type.Size) > len(data) {
			continue
		}
		fieldData := data[field.Offset : field.Offset+field.Type.Size]
		nestedResult[field.Name] = parseFieldValue(fieldData, field.Type)
	}
	return nestedResult
}

// parseSimpleType parses simple data types. Returns nil if type cannot be parsed.
func parseSimpleType(data []byte, baseType symbols.DataType) interface{} {
	switch baseType {
	case symbols.DataTypeBool:
		return parseBool(data)
	case symbols.DataTypeInt8, symbols.DataTypeUInt8:
		return parseInt8Types(data, baseType)
	case symbols.DataTypeInt16, symbols.DataTypeUInt16:
		return parseInt16Types(data, baseType)
	case symbols.DataTypeInt32, symbols.DataTypeUInt32:
		return parseInt32Types(data, baseType)
	case symbols.DataTypeInt64, symbols.DataTypeUInt64:
		return parseInt64Types(data, baseType)
	case symbols.DataTypeReal32:
		return parseReal32(data)
	case symbols.DataTypeReal64:
		return parseReal64(data)
	case symbols.DataTypeString:
		return parseString(data)
	}
	return nil
}

// parseBool parses a boolean value.
func parseBool(data []byte) interface{} {
	if len(data) >= 1 {
		return data[0] != 0
	}
	return nil
}

// parseInt8Types parses 8-bit integer types.
func parseInt8Types(data []byte, baseType symbols.DataType) interface{} {
	if len(data) < 1 {
		return nil
	}
	if baseType == symbols.DataTypeInt8 {
		return int8(data[0])
	}
	return uint8(data[0])
}

// parseInt16Types parses 16-bit integer types.
func parseInt16Types(data []byte, baseType symbols.DataType) interface{} {
	if len(data) < 2 {
		return nil
	}
	val := binary.LittleEndian.Uint16(data)
	if baseType == symbols.DataTypeInt16 {
		return int16(val)
	}
	return val
}

// parseInt32Types parses 32-bit integer types.
func parseInt32Types(data []byte, baseType symbols.DataType) interface{} {
	if len(data) < 4 {
		return nil
	}
	val := binary.LittleEndian.Uint32(data)
	if baseType == symbols.DataTypeInt32 {
		return int32(val)
	}
	return val
}

// parseInt64Types parses 64-bit integer types.
func parseInt64Types(data []byte, baseType symbols.DataType) interface{} {
	if len(data) < 8 {
		return nil
	}
	val := binary.LittleEndian.Uint64(data)
	if baseType == symbols.DataTypeInt64 {
		return int64(val)
	}
	return val
}

// parseReal32 parses a 32-bit floating point value.
func parseReal32(data []byte) interface{} {
	if len(data) >= 4 {
		bits := binary.LittleEndian.Uint32(data)
		return math.Float32frombits(bits)
	}
	return nil
}

// parseReal64 parses a 64-bit floating point value.
func parseReal64(data []byte) interface{} {
	if len(data) >= 8 {
		bits := binary.LittleEndian.Uint64(data)
		return math.Float64frombits(bits)
	}
	return nil
}

// parseString parses a null-terminated string.
func parseString(data []byte) interface{} {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}
