// Package symbols implements symbol table parsing and caching for TwinCAT 3.
package symbols

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// DataType represents TwinCAT data types.
type DataType uint32

const (
	DataTypeVoid        DataType = 0
	DataTypeInt8        DataType = 16
	DataTypeUInt8       DataType = 17
	DataTypeInt16       DataType = 2
	DataTypeUInt16      DataType = 18
	DataTypeInt32       DataType = 3
	DataTypeUInt32      DataType = 19
	DataTypeInt64       DataType = 20
	DataTypeUInt64      DataType = 21
	DataTypeReal32      DataType = 4
	DataTypeReal64      DataType = 5
	DataTypeBool        DataType = 33
	DataTypeString      DataType = 30
	DataTypeWString     DataType = 31
	DataTypeReal80      DataType = 32
	DataTypeBit         DataType = 1
	DataTypeTime        DataType = 36
	DataTypeTimeOfDay   DataType = 37
	DataTypeDate        DataType = 38
	DataTypeDateAndTime DataType = 39
)

// TypeInfo represents parsed type information.
type TypeInfo struct {
	Name      string      // Type name
	BaseType  DataType    // Base data type
	Size      uint32      // Size in bytes
	ArrayDims []uint32    // Array dimensions
	IsArray   bool        // True if array type
	IsStruct  bool        // True if struct type
	Fields    []FieldInfo // Struct fields
	Comment   string      // Type comment
}

// FieldInfo represents a struct field.
type FieldInfo struct {
	Name      string
	Offset    uint32
	Type      TypeInfo
	BitOffset uint8
	BitSize   uint8
}

// Symbol represents a parsed PLC symbol.
type Symbol struct {
	Name        string
	Type        TypeInfo
	IndexGroup  uint32
	IndexOffset uint32
	Size        uint32
	Flags       uint32
	Comment     string
}

// ParseSymbolTable parses raw symbol upload data.
func ParseSymbolTable(data []byte) ([]Symbol, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("symbol data is empty")
	}

	var symbols []Symbol
	offset := 0

	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		entryLength := binary.LittleEndian.Uint32(data[offset : offset+4])
		if entryLength == 0 {
			break
		}

		if offset+int(entryLength) > len(data) {
			return nil, fmt.Errorf("invalid entry length %d at offset %d", entryLength, offset)
		}

		entryData := data[offset : offset+int(entryLength)]
		symbol, err := parseSymbolEntry(entryData)
		if err != nil {
			return nil, fmt.Errorf("parse symbol at offset %d: %w", offset, err)
		}

		symbols = append(symbols, symbol)
		offset += int(entryLength)
	}

	return symbols, nil
}

func parseSymbolEntry(data []byte) (Symbol, error) {
	if len(data) < 30 {
		return Symbol{}, fmt.Errorf("symbol entry too short: %d bytes", len(data))
	}

	symbol := Symbol{
		IndexGroup:  binary.LittleEndian.Uint32(data[4:8]),
		IndexOffset: binary.LittleEndian.Uint32(data[8:12]),
		Size:        binary.LittleEndian.Uint32(data[12:16]),
		Flags:       binary.LittleEndian.Uint32(data[20:24]),
	}

	dataTypeID := binary.LittleEndian.Uint32(data[16:20])
	nameLength := binary.LittleEndian.Uint16(data[24:26])
	typeLength := binary.LittleEndian.Uint16(data[26:28])
	commentLength := binary.LittleEndian.Uint16(data[28:30])

	stringOffset := 30
	if stringOffset+int(nameLength) > len(data) {
		return Symbol{}, fmt.Errorf("invalid name length")
	}
	symbol.Name = parseString(data[stringOffset : stringOffset+int(nameLength)+1])
	stringOffset += int(nameLength) + 1

	if stringOffset+int(typeLength) > len(data) {
		return Symbol{}, fmt.Errorf("invalid type length")
	}
	typeName := parseString(data[stringOffset : stringOffset+int(typeLength)+1])
	stringOffset += int(typeLength) + 1

	if stringOffset+int(commentLength) > len(data) {
		return Symbol{}, fmt.Errorf("invalid comment length")
	}
	symbol.Comment = parseString(data[stringOffset : stringOffset+int(commentLength)+1])

	symbol.Type = parseTypeInfo(typeName, DataType(dataTypeID), symbol.Size)

	return symbol, nil
}

func parseTypeInfo(typeName string, dataTypeID DataType, size uint32) TypeInfo {
	typeInfo := TypeInfo{
		Name:     typeName,
		BaseType: dataTypeID,
		Size:     size,
	}

	if strings.Contains(typeName, "ARRAY") {
		typeInfo.IsArray = true
		typeInfo.ArrayDims = parseArrayDimensions(typeName)
	}

	if dataTypeID == 65 || !isSimpleType(dataTypeID) {
		typeInfo.IsStruct = true
	}

	return typeInfo
}

func parseArrayDimensions(typeName string) []uint32 {
	var dims []uint32

	start := strings.Index(typeName, "[")
	end := strings.Index(typeName, "]")

	if start == -1 || end == -1 {
		return dims
	}

	rangeStr := typeName[start+1 : end]
	ranges := strings.Split(rangeStr, ",")

	for _, r := range ranges {
		parts := strings.Split(strings.TrimSpace(r), "..")
		if len(parts) == 2 {
			var low, high uint32
			fmt.Sscanf(parts[0], "%d", &low)
			fmt.Sscanf(parts[1], "%d", &high)
			dims = append(dims, high-low+1)
		}
	}

	return dims
}

func parseString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

func isSimpleType(dt DataType) bool {
	switch dt {
	case DataTypeInt8, DataTypeUInt8, DataTypeInt16, DataTypeUInt16,
		DataTypeInt32, DataTypeUInt32, DataTypeInt64, DataTypeUInt64,
		DataTypeReal32, DataTypeReal64, DataTypeBool, DataTypeBit:
		return true
	default:
		return false
	}
}

func (dt DataType) String() string {
	switch dt {
	case DataTypeInt8:
		return "SINT"
	case DataTypeUInt8:
		return "USINT"
	case DataTypeInt16:
		return "INT"
	case DataTypeUInt16:
		return "UINT"
	case DataTypeInt32:
		return "DINT"
	case DataTypeUInt32:
		return "UDINT"
	case DataTypeInt64:
		return "LINT"
	case DataTypeUInt64:
		return "ULINT"
	case DataTypeReal32:
		return "REAL"
	case DataTypeReal64:
		return "LREAL"
	case DataTypeBool:
		return "BOOL"
	case DataTypeString:
		return "STRING"
	default:
		return fmt.Sprintf("TYPE_%d", dt)
	}
}
