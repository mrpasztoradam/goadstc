package goadstc

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"
	"unicode/utf16"
)

// Type-safe read methods for common TwinCAT types

// ReadBool reads a BOOL value from a symbol by name.
func (c *Client) ReadBool(ctx context.Context, symbolName string) (bool, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return false, err
	}
	if len(data) < 1 {
		return false, fmt.Errorf("insufficient data: expected at least 1 byte, got %d", len(data))
	}
	return data[0] != 0, nil
}

// ReadInt8 reads an INT8/SINT value from a symbol by name.
func (c *Client) ReadInt8(ctx context.Context, symbolName string) (int8, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 1 {
		return 0, fmt.Errorf("insufficient data: expected at least 1 byte, got %d", len(data))
	}
	return int8(data[0]), nil
}

// ReadUint8 reads a UINT8/USINT/BYTE value from a symbol by name.
func (c *Client) ReadUint8(ctx context.Context, symbolName string) (uint8, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 1 {
		return 0, fmt.Errorf("insufficient data: expected at least 1 byte, got %d", len(data))
	}
	return uint8(data[0]), nil
}

// ReadInt16 reads an INT16/INT value from a symbol by name.
func (c *Client) ReadInt16(ctx context.Context, symbolName string) (int16, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, fmt.Errorf("insufficient data: expected at least 2 bytes, got %d", len(data))
	}
	return int16(binary.LittleEndian.Uint16(data)), nil
}

// ReadUint16 reads a UINT16/UINT/WORD value from a symbol by name.
func (c *Client) ReadUint16(ctx context.Context, symbolName string) (uint16, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, fmt.Errorf("insufficient data: expected at least 2 bytes, got %d", len(data))
	}
	return binary.LittleEndian.Uint16(data), nil
}

// ReadInt32 reads an INT32/DINT value from a symbol by name.
func (c *Client) ReadInt32(ctx context.Context, symbolName string) (int32, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 4 {
		return 0, fmt.Errorf("insufficient data: expected at least 4 bytes, got %d", len(data))
	}
	return int32(binary.LittleEndian.Uint32(data)), nil
}

// ReadUint32 reads a UINT32/UDINT/DWORD value from a symbol by name.
func (c *Client) ReadUint32(ctx context.Context, symbolName string) (uint32, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 4 {
		return 0, fmt.Errorf("insufficient data: expected at least 4 bytes, got %d", len(data))
	}
	return binary.LittleEndian.Uint32(data), nil
}

// ReadInt64 reads an INT64/LINT value from a symbol by name.
func (c *Client) ReadInt64(ctx context.Context, symbolName string) (int64, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("insufficient data: expected at least 8 bytes, got %d", len(data))
	}
	return int64(binary.LittleEndian.Uint64(data)), nil
}

// ReadUint64 reads a UINT64/ULINT/LWORD value from a symbol by name.
func (c *Client) ReadUint64(ctx context.Context, symbolName string) (uint64, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("insufficient data: expected at least 8 bytes, got %d", len(data))
	}
	return binary.LittleEndian.Uint64(data), nil
}

// ReadFloat32 reads a REAL/FLOAT value from a symbol by name.
func (c *Client) ReadFloat32(ctx context.Context, symbolName string) (float32, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 4 {
		return 0, fmt.Errorf("insufficient data: expected at least 4 bytes, got %d", len(data))
	}
	bits := binary.LittleEndian.Uint32(data)
	return math.Float32frombits(bits), nil
}

// ReadFloat64 reads an LREAL/DOUBLE value from a symbol by name.
func (c *Client) ReadFloat64(ctx context.Context, symbolName string) (float64, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("insufficient data: expected at least 8 bytes, got %d", len(data))
	}
	bits := binary.LittleEndian.Uint64(data)
	return math.Float64frombits(bits), nil
}

// ReadString reads a STRING value from a symbol by name.
// TwinCAT strings are null-terminated and may have a fixed buffer size.
// Returns the string up to the first null byte.
func (c *Client) ReadString(ctx context.Context, symbolName string) (string, error) {
	data, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return "", err
	}
	// Find null terminator
	nullIndex := 0
	for nullIndex < len(data) && data[nullIndex] != 0 {
		nullIndex++
	}
	return string(data[:nullIndex]), nil
}

// ReadTime reads a TIME value from a symbol and returns it as time.Duration.
// TIME is stored as a 32-bit signed integer representing milliseconds.
func (c *Client) ReadTime(ctx context.Context, symbolName string) (time.Duration, error) {
	val, err := c.ReadInt32(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	return time.Duration(val) * time.Millisecond, nil
}

// ReadDate reads a DATE value from a symbol and returns it as time.Time.
// DATE is stored as a 32-bit unsigned integer representing seconds since 1970-01-01.
func (c *Client) ReadDate(ctx context.Context, symbolName string) (time.Time, error) {
	val, err := c.ReadUint32(ctx, symbolName)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(val), 0), nil
}

// ReadTimeOfDay reads a TIME_OF_DAY value from a symbol and returns it as time.Duration.
// TIME_OF_DAY is stored as a 32-bit unsigned integer representing milliseconds since midnight.
func (c *Client) ReadTimeOfDay(ctx context.Context, symbolName string) (time.Duration, error) {
	val, err := c.ReadUint32(ctx, symbolName)
	if err != nil {
		return 0, err
	}
	return time.Duration(val) * time.Millisecond, nil
}

// ReadDateAndTime reads a DATE_AND_TIME value from a symbol and returns it as time.Time.
// DATE_AND_TIME is stored as a 32-bit unsigned integer representing seconds since 1970-01-01.
func (c *Client) ReadDateAndTime(ctx context.Context, symbolName string) (time.Time, error) {
	val, err := c.ReadUint32(ctx, symbolName)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(val), 0), nil
}

// ReadWString reads a WSTRING (wide string, UTF-16LE) value from a symbol.
// Returns the string as UTF-8.
func (c *Client) ReadWString(ctx context.Context, symbolName string) (string, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return "", err
	}

	indexGroup, indexOffset, size, err := c.resolveArraySymbol(ctx, symbolName)
	if err != nil {
		return "", fmt.Errorf("read wstring %q: %w", symbolName, err)
	}

	data, err := c.Read(ctx, indexGroup, indexOffset, size)
	if err != nil {
		return "", err
	}

	// Convert UTF-16LE to UTF-8
	// Find the null terminator (2 bytes of zeros)
	var length int
	for i := 0; i < len(data)-1; i += 2 {
		if data[i] == 0 && data[i+1] == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		length = len(data)
	}

	// Decode UTF-16LE
	uint16s := make([]uint16, length/2)
	for i := 0; i < length/2; i++ {
		uint16s[i] = uint16(data[i*2]) | uint16(data[i*2+1])<<8
	}

	return string(utf16.Decode(uint16s)), nil
}

// Type-safe write methods for common TwinCAT types

// WriteBool writes a BOOL value to a symbol by name.
func (c *Client) WriteBool(ctx context.Context, symbolName string, value bool) error {
	data := make([]byte, 1)
	if value {
		data[0] = 1
	}
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteInt8 writes an INT8/SINT value to a symbol by name.
func (c *Client) WriteInt8(ctx context.Context, symbolName string, value int8) error {
	data := []byte{byte(value)}
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteUint8 writes a UINT8/USINT/BYTE value to a symbol by name.
func (c *Client) WriteUint8(ctx context.Context, symbolName string, value uint8) error {
	data := []byte{value}
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteInt16 writes an INT16/INT value to a symbol by name.
func (c *Client) WriteInt16(ctx context.Context, symbolName string, value int16) error {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, uint16(value))
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteUint16 writes a UINT16/UINT/WORD value to a symbol by name.
func (c *Client) WriteUint16(ctx context.Context, symbolName string, value uint16) error {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, value)
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteInt32 writes an INT32/DINT value to a symbol by name.
func (c *Client) WriteInt32(ctx context.Context, symbolName string, value int32) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(value))
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteUint32 writes a UINT32/UDINT/DWORD value to a symbol by name.
func (c *Client) WriteUint32(ctx context.Context, symbolName string, value uint32) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteInt64 writes an INT64/LINT value to a symbol by name.
func (c *Client) WriteInt64(ctx context.Context, symbolName string, value int64) error {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(value))
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteUint64 writes a UINT64/ULINT/LWORD value to a symbol by name.
func (c *Client) WriteUint64(ctx context.Context, symbolName string, value uint64) error {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, value)
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteFloat32 writes a REAL/FLOAT value to a symbol by name.
func (c *Client) WriteFloat32(ctx context.Context, symbolName string, value float32) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(value))
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteFloat64 writes an LREAL/DOUBLE value to a symbol by name.
func (c *Client) WriteFloat64(ctx context.Context, symbolName string, value float64) error {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, math.Float64bits(value))
	return c.WriteSymbol(ctx, symbolName, data)
}

// WriteString writes a STRING value to a symbol by name.
// TwinCAT strings have a fixed buffer size. The value is null-terminated
// and padded with zeros to fill the buffer.
func (c *Client) WriteString(ctx context.Context, symbolName string, value string) error {
	// First, resolve the symbol to get its size (the string buffer size)
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return err
	}

	indexGroup, indexOffset, size, err := c.resolveArraySymbol(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("write string %q: %w", symbolName, err)
	}

	// Create buffer with the string's allocated size
	data := make([]byte, size)

	// Copy string bytes (up to size-1 to leave room for null terminator)
	maxLen := int(size) - 1
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	copy(data, []byte(value))
	// data is already zero-filled, so null terminator is implicit

	return c.Write(ctx, indexGroup, indexOffset, data)
}

// WriteTime writes a time.Duration value to a TIME symbol.
// TIME is stored as a 32-bit signed integer representing milliseconds.
func (c *Client) WriteTime(ctx context.Context, symbolName string, value time.Duration) error {
	ms := int32(value / time.Millisecond)
	return c.WriteInt32(ctx, symbolName, ms)
}

// WriteDate writes a time.Time value to a DATE symbol.
// DATE is stored as a 32-bit unsigned integer representing seconds since 1970-01-01.
func (c *Client) WriteDate(ctx context.Context, symbolName string, value time.Time) error {
	secs := uint32(value.Unix())
	return c.WriteUint32(ctx, symbolName, secs)
}

// WriteTimeOfDay writes a time.Duration value to a TIME_OF_DAY symbol.
// TIME_OF_DAY is stored as a 32-bit unsigned integer representing milliseconds since midnight.
func (c *Client) WriteTimeOfDay(ctx context.Context, symbolName string, value time.Duration) error {
	ms := uint32(value / time.Millisecond)
	return c.WriteUint32(ctx, symbolName, ms)
}

// WriteDateAndTime writes a time.Time value to a DATE_AND_TIME symbol.
// DATE_AND_TIME is stored as a 32-bit unsigned integer representing seconds since 1970-01-01.
func (c *Client) WriteDateAndTime(ctx context.Context, symbolName string, value time.Time) error {
	secs := uint32(value.Unix())
	return c.WriteUint32(ctx, symbolName, secs)
}

// WriteWString writes a string value to a WSTRING symbol.
// The string is converted from UTF-8 to UTF-16LE.
// WSTRING has a fixed buffer size, and the value is null-terminated and padded with zeros.
func (c *Client) WriteWString(ctx context.Context, symbolName string, value string) error {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return err
	}

	indexGroup, indexOffset, size, err := c.resolveArraySymbol(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("write wstring %q: %w", symbolName, err)
	}

	// Create buffer with the string's allocated size
	data := make([]byte, size)

	// Encode string to UTF-16LE
	uint16s := utf16.Encode([]rune(value))

	// Calculate max number of UTF-16 code units that fit (leave room for null terminator)
	maxUnits := (int(size) / 2) - 1
	if len(uint16s) > maxUnits {
		uint16s = uint16s[:maxUnits]
	}

	// Write UTF-16LE bytes
	for i, u := range uint16s {
		data[i*2] = byte(u)
		data[i*2+1] = byte(u >> 8)
	}
	// data is already zero-filled, so null terminator is implicit

	return c.Write(ctx, indexGroup, indexOffset, data)
}
