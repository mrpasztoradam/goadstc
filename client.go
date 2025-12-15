// Package goadstc provides a Go client library for TwinCAT ADS/AMS communication over TCP.
package goadstc

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/ams"
	"github.com/mrpasztoradam/goadstc/internal/symbols"
	"github.com/mrpasztoradam/goadstc/internal/transport"
)

// Client represents an ADS client connection.
type Client struct {
	conn            *transport.Conn
	targetNetID     ams.NetID
	targetPort      ams.Port
	sourceNetID     ams.NetID
	sourcePort      ams.Port
	subscriptions   map[uint32]*Subscription
	subscriptionsMu sync.RWMutex
	symbolTable     *symbols.Table
	symbolTableMu   sync.RWMutex
	typeRegistry    *symbols.TypeRegistry
	typeRegistryMu  sync.RWMutex
}

// DeviceInfo represents device information returned by ReadDeviceInfo.
type DeviceInfo struct {
	Name         string
	MajorVersion uint8
	MinorVersion uint8
	VersionBuild uint16
}

// DeviceState represents the state of an ADS device.
type DeviceState struct {
	ADSState    ads.ADSState
	DeviceState uint16
}

// Option is a functional option for configuring a Client.
type Option func(*clientConfig) error

type clientConfig struct {
	address     string
	targetNetID ams.NetID
	targetPort  ams.Port
	sourceNetID ams.NetID
	sourcePort  ams.Port
	timeout     time.Duration
}

// WithTarget sets the target TCP address (required).
func WithTarget(address string) Option {
	return func(c *clientConfig) error {
		if address == "" {
			return fmt.Errorf("goadstc: target address cannot be empty")
		}
		c.address = address
		return nil
	}
}

// WithAMSNetID sets the target AMS NetID (required).
func WithAMSNetID(netID ams.NetID) Option {
	return func(c *clientConfig) error {
		c.targetNetID = netID
		return nil
	}
}

// WithAMSPort sets the target AMS port (optional, defaults to 851).
func WithAMSPort(port ams.Port) Option {
	return func(c *clientConfig) error {
		c.targetPort = port
		return nil
	}
}

// WithSourceNetID sets the source AMS NetID (optional).
func WithSourceNetID(netID ams.NetID) Option {
	return func(c *clientConfig) error {
		c.sourceNetID = netID
		return nil
	}
}

// WithSourcePort sets the source AMS port (optional).
func WithSourcePort(port ams.Port) Option {
	return func(c *clientConfig) error {
		c.sourcePort = port
		return nil
	}
}

// WithTimeout sets the timeout for requests (optional).
func WithTimeout(timeout time.Duration) Option {
	return func(c *clientConfig) error {
		if timeout <= 0 {
			return fmt.Errorf("goadstc: timeout must be positive")
		}
		c.timeout = timeout
		return nil
	}
}

// New creates a new ADS client with the given options.
func New(opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		targetPort: ams.PortPLCRuntime1,
		sourcePort: 32905,
		timeout:    5 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	if cfg.address == "" {
		return nil, fmt.Errorf("goadstc: target address is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	conn, err := transport.Dial(ctx, cfg.address, cfg.timeout)
	if err != nil {
		return nil, fmt.Errorf("goadstc: connection failed: %w", err)
	}

	client := &Client{
		conn:          conn,
		targetNetID:   cfg.targetNetID,
		targetPort:    cfg.targetPort,
		sourceNetID:   cfg.sourceNetID,
		sourcePort:    cfg.sourcePort,
		subscriptions: make(map[uint32]*Subscription),
		symbolTable:   symbols.NewTable(),
		typeRegistry:  symbols.NewTypeRegistry(),
	}

	// Set up notification handler
	conn.SetNotificationHandler(client.handleNotification)

	return client, nil
}

// Close closes the client connection and all active subscriptions.
func (c *Client) Close() error {
	// Close all subscriptions
	c.subscriptionsMu.Lock()
	subs := make([]*Subscription, 0, len(c.subscriptions))
	for _, sub := range c.subscriptions {
		subs = append(subs, sub)
	}
	c.subscriptionsMu.Unlock()

	for _, sub := range subs {
		sub.Close()
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) sendRequest(ctx context.Context, commandID ads.CommandID, reqData []byte) (*ams.Packet, error) {
	invokeID := c.conn.NextInvokeID()
	reqPacket := ams.NewRequestPacket(
		c.targetNetID, c.targetPort,
		c.sourceNetID, c.sourcePort,
		uint16(commandID), invokeID, reqData,
	)

	respPacket, err := c.conn.SendRequest(ctx, reqPacket)
	if err != nil {
		return nil, err
	}

	if respPacket.Header.ErrorCode != 0 {
		return nil, ads.Error(respPacket.Header.ErrorCode)
	}

	return respPacket, nil
}

// GetSymbolHandle retrieves a handle for the given symbol name.
// The handle can be used with Read/Write operations using the handle's IndexGroup/IndexOffset.
// Handles should be released with ReleaseSymbolHandle when no longer needed.
func (c *Client) GetSymbolHandle(ctx context.Context, symbolName string) (uint32, error) {
	// Prepare symbol name as null-terminated string
	nameBytes := []byte(symbolName)
	nameBytes = append(nameBytes, 0) // Add null terminator

	// Use ReadWrite command with ADSIGRP_SYM_HNDBYNAME (0xF003)
	readData, err := c.ReadWrite(ctx, 0xF003, 0, 4, nameBytes)
	if err != nil {
		return 0, fmt.Errorf("get symbol handle for %q: %w", symbolName, err)
	}

	if len(readData) < 4 {
		return 0, fmt.Errorf("invalid symbol handle response: expected 4 bytes, got %d", len(readData))
	}

	var resp ads.GetSymbolHandleByNameResponse
	if err := resp.UnmarshalBinary(readData); err != nil {
		return 0, fmt.Errorf("parse symbol handle response: %w", err)
	}

	return resp.Handle, nil
}

// ReleaseSymbolHandle releases a previously acquired symbol handle.
func (c *Client) ReleaseSymbolHandle(ctx context.Context, handle uint32) error {
	// Prepare handle as 4-byte data
	handleData := make([]byte, 4)
	binary.LittleEndian.PutUint32(handleData, handle)

	// Use Write command with ADSIGRP_SYM_RELEASEHND (0xF006)
	if err := c.Write(ctx, 0xF006, 0, handleData); err != nil {
		return fmt.Errorf("release symbol handle %d: %w", handle, err)
	}

	return nil
}

// GetSymbolUploadInfo retrieves information about the PLC symbol table.
// Returns the number of symbols and total size of symbol data.
func (c *Client) GetSymbolUploadInfo(ctx context.Context) (symbolCount, symbolLength uint32, err error) {
	// Use Read command with ADSIGRP_SYM_UPLOADINFO2 (0xF00C)
	readData, err := c.Read(ctx, 0xF00C, 0, 0x30) // 48 bytes for upload info
	if err != nil {
		return 0, 0, fmt.Errorf("get symbol upload info: %w", err)
	}

	if len(readData) < 8 {
		return 0, 0, fmt.Errorf("invalid upload info response: expected at least 8 bytes, got %d", len(readData))
	}

	var resp ads.SymbolUploadInfoResponse
	if err := resp.UnmarshalBinary(readData); err != nil {
		return 0, 0, fmt.Errorf("parse symbol upload info: %w", err)
	}

	return resp.SymbolCount, resp.SymbolLength, nil
}

// UploadSymbolTable downloads the complete symbol table from the PLC.
// The returned data is in raw TwinCAT format and needs parsing.
func (c *Client) UploadSymbolTable(ctx context.Context) ([]byte, error) {
	// First get the size
	_, symbolLength, err := c.GetSymbolUploadInfo(ctx)
	if err != nil {
		return nil, err
	}

	if symbolLength == 0 {
		return nil, fmt.Errorf("symbol table is empty")
	}

	// Create a context with extended timeout for large symbol table downloads
	// Symbol tables can be large (several MB) and take time to transfer
	uploadCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		// Extend the deadline by 30 seconds for symbol upload
		extendedDeadline := deadline.Add(30 * time.Second)
		var cancel context.CancelFunc
		uploadCtx, cancel = context.WithDeadline(context.Background(), extendedDeadline)
		defer cancel()
	} else {
		// No existing deadline, create one with 30 seconds
		var cancel context.CancelFunc
		uploadCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Use Read command with ADSIGRP_SYM_UPLOAD (0xF00B)
	// Request the exact size reported by the PLC
	readData, err := c.Read(uploadCtx, 0xF00B, 0, symbolLength)
	if err != nil {
		return nil, fmt.Errorf("upload symbol table: %w", err)
	}

	return readData, nil
}

// RefreshSymbols downloads and parses the symbol table from the PLC.
// This method should be called before using symbol-based operations.
// It can be called multiple times to refresh the cache if the PLC program changes.
func (c *Client) RefreshSymbols(ctx context.Context) error {
	data, err := c.UploadSymbolTable(ctx)
	if err != nil {
		return fmt.Errorf("refresh symbols: %w", err)
	}

	c.symbolTableMu.Lock()
	defer c.symbolTableMu.Unlock()

	if err := c.symbolTable.Load(data); err != nil {
		return fmt.Errorf("load symbols: %w", err)
	}

	return nil
}

// ensureSymbolsLoaded automatically loads symbols if not already loaded.
func (c *Client) ensureSymbolsLoaded(ctx context.Context) error {
	c.symbolTableMu.RLock()
	loaded := c.symbolTable.IsLoaded()
	c.symbolTableMu.RUnlock()

	if !loaded {
		return c.RefreshSymbols(ctx)
	}
	return nil
}

// GetSymbol retrieves symbol information by name.
func (c *Client) GetSymbol(name string) (*symbols.Symbol, error) {
	c.symbolTableMu.RLock()
	defer c.symbolTableMu.RUnlock()

	return c.symbolTable.Get(name)
}

// ListSymbols returns all symbols in the cache.
// Calls RefreshSymbols automatically if symbols not loaded.
func (c *Client) ListSymbols(ctx context.Context) ([]*symbols.Symbol, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, err
	}

	c.symbolTableMu.RLock()
	defer c.symbolTableMu.RUnlock()

	return c.symbolTable.List()
}

// FindSymbols searches for symbols matching the pattern (case-insensitive substring).
func (c *Client) FindSymbols(ctx context.Context, pattern string) ([]*symbols.Symbol, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, err
	}

	c.symbolTableMu.RLock()
	defer c.symbolTableMu.RUnlock()

	return c.symbolTable.Find(pattern)
}

// parseArrayAccess parses array notation like "MAIN.myArray[5]" or "MAIN.matrix[2][3]"
// Returns: base symbol name, array indices, error
func parseArrayAccess(symbolName string) (string, []int, error) {
	// Check if there's array notation
	if !strings.Contains(symbolName, "[") {
		return symbolName, nil, nil
	}

	// Find the base name (everything before first '[')
	firstBracket := strings.Index(symbolName, "[")
	baseName := symbolName[:firstBracket]

	// Parse indices
	var indices []int
	remainder := symbolName[firstBracket:]

	for len(remainder) > 0 && remainder[0] == '[' {
		closeBracket := strings.Index(remainder, "]")
		if closeBracket == -1 {
			return "", nil, fmt.Errorf("invalid array notation: missing ']'")
		}

		indexStr := remainder[1:closeBracket]
		var idx int
		n, err := fmt.Sscanf(indexStr, "%d", &idx)
		if err != nil || n != 1 {
			return "", nil, fmt.Errorf("invalid array index: %q", indexStr)
		}
		indices = append(indices, idx)

		remainder = remainder[closeBracket+1:]
	}

	// If there's a remainder (like ".fieldName"), add it to the base name
	if len(remainder) > 0 {
		baseName = baseName + remainder
	}

	return baseName, indices, nil
}

// extractArrayElementType extracts the element type from an array type name.
// e.g., "ARRAY [0..9] OF INT" -> "INT"
// e.g., "ARRAY [0..4] OF TestSt" -> "TestSt"
func extractArrayElementType(arrayTypeName string) (string, bool) {
	ofIndex := strings.Index(arrayTypeName, " OF ")
	if ofIndex == -1 {
		return "", false
	}
	elementType := strings.TrimSpace(arrayTypeName[ofIndex+4:])
	return elementType, true
}

// resolveArraySymbol resolves array element access and returns the adjusted symbol info.
// For "MAIN.myArray[5]", it returns the base symbol with IndexOffset adjusted for element 5.
// For "MAIN.myArray[5].field", it resolves both array and struct field offsets.
func (c *Client) resolveArraySymbol(ctx context.Context, symbolName string) (indexGroup, indexOffset, size uint32, err error) {
	// First check if there's a struct field path (e.g., "MAIN.aStruct[0].uiTest")
	// We need to separate: "MAIN.aStruct" + "[0]" + ".uiTest"
	var arrayBase string
	var structField string

	// Find if there's a dot after a closing bracket (indicates struct field after array)
	firstBracket := strings.Index(symbolName, "[")
	if firstBracket != -1 {
		closeBracket := strings.Index(symbolName[firstBracket:], "]")
		if closeBracket != -1 {
			afterBracket := firstBracket + closeBracket + 1
			if afterBracket < len(symbolName) && symbolName[afterBracket] == '.' {
				// We have struct field access after array: split it
				arrayPart := symbolName[:afterBracket]
				structField = symbolName[afterBracket+1:] // Skip the dot

				baseName, indices, err := parseArrayAccess(arrayPart)
				if err != nil {
					return 0, 0, 0, err
				}
				arrayBase = baseName

				// Now handle as "arrayBase[index].field"
				symbol, err := c.GetSymbol(arrayBase)
				if err != nil {
					return 0, 0, 0, fmt.Errorf("resolve symbol %q: %w", arrayBase, err)
				}

				if len(indices) > 1 {
					return 0, 0, 0, fmt.Errorf("multi-dimensional arrays not yet supported")
				}

				// Get array element type
				elementTypeName, isArray := extractArrayElementType(symbol.Type.Name)
				if !isArray {
					return 0, 0, 0, fmt.Errorf("%q is not an array type", arrayBase)
				}

				// Get element type info
				elementTypeInfo, err := c.getOrFetchTypeInfo(ctx, elementTypeName)
				if err != nil {
					return 0, 0, 0, fmt.Errorf("get element type info for %q: %w", elementTypeName, err)
				}

				// Calculate array element offset
				elementOffset := uint32(indices[0]) * elementTypeInfo.Size

				// Now find the struct field offset within the element
				fieldInfo, found := findFieldInType(elementTypeInfo, structField)
				if !found {
					return 0, 0, 0, fmt.Errorf("field %q not found in type %q", structField, elementTypeName)
				}

				return symbol.IndexGroup, symbol.IndexOffset + elementOffset + fieldInfo.Offset, fieldInfo.Type.Size, nil
			}
		}
	}

	// No struct field after array, use normal array resolution
	baseName, indices, err := parseArrayAccess(symbolName)
	if err != nil {
		return 0, 0, 0, err
	}

	// Get the base symbol
	symbol, err := c.GetSymbol(baseName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("resolve symbol %q: %w", baseName, err)
	}

	// If no array access, return as-is
	if len(indices) == 0 {
		return symbol.IndexGroup, symbol.IndexOffset, symbol.Size, nil
	}

	// Calculate offset for array access
	if len(indices) > 1 {
		return 0, 0, 0, fmt.Errorf("multi-dimensional arrays not yet supported")
	}

	// For arrays, extract the element type from the array type name
	// e.g., "ARRAY [0..9] OF INT" -> element type is "INT"
	elementTypeName, isArray := extractArrayElementType(symbol.Type.Name)
	if !isArray {
		// Not an array type, use the symbol's type directly (shouldn't happen with valid array access)
		elementTypeName = symbol.Type.Name
	}

	// Get element type info to determine element size
	elementTypeInfo, err := c.getOrFetchTypeInfo(ctx, elementTypeName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get element type info for %q: %w", elementTypeName, err)
	}

	elementSize := elementTypeInfo.Size
	offset := uint32(indices[0]) * elementSize

	return symbol.IndexGroup, symbol.IndexOffset + offset, elementSize, nil
}

// findFieldInType searches for a field in a type's field list.
func findFieldInType(typeInfo symbols.TypeInfo, fieldName string) (symbols.FieldInfo, bool) {
	for _, field := range typeInfo.Fields {
		if field.Name == fieldName {
			return field, true
		}
	}
	return symbols.FieldInfo{}, false
}

// ReadSymbol reads data from a PLC symbol by name.
// Supports array element access using bracket notation: "MAIN.myArray[5]"
// Automatically loads symbol table on first call.
func (c *Client) ReadSymbol(ctx context.Context, symbolName string) ([]byte, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, err
	}

	indexGroup, indexOffset, size, err := c.resolveArraySymbol(ctx, symbolName)
	if err != nil {
		return nil, fmt.Errorf("read symbol %q: %w", symbolName, err)
	}

	return c.Read(ctx, indexGroup, indexOffset, size)
}

// WriteSymbol writes data to a PLC symbol by name.
// Supports array element access using bracket notation: "MAIN.myArray[5]"
// Automatically loads symbol table on first call.
func (c *Client) WriteSymbol(ctx context.Context, symbolName string, data []byte) error {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return err
	}

	indexGroup, indexOffset, size, err := c.resolveArraySymbol(ctx, symbolName)
	if err != nil {
		return fmt.Errorf("write symbol %q: %w", symbolName, err)
	}

	if uint32(len(data)) != size {
		return fmt.Errorf("write symbol %q: data size mismatch (expected %d bytes, got %d)",
			symbolName, size, len(data))
	}

	return c.Write(ctx, indexGroup, indexOffset, data)
}

// ReadDeviceInfo reads the device name and version.
func (c *Client) ReadDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	req := ads.ReadDeviceInfoRequest{}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdReadDeviceInfo, reqData)
	if err != nil {
		return nil, err
	}

	var resp ads.ReadDeviceInfoResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, ads.Error(resp.Result)
	}

	return &DeviceInfo{
		Name:         resp.DeviceName,
		MajorVersion: resp.MajorVersion,
		MinorVersion: resp.MinorVersion,
		VersionBuild: resp.VersionBuild,
	}, nil
}

// Read reads data from the ADS device.
func (c *Client) Read(ctx context.Context, indexGroup, indexOffset, length uint32) ([]byte, error) {
	req := ads.ReadRequest{
		IndexGroup:  indexGroup,
		IndexOffset: indexOffset,
		Length:      length,
	}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdRead, reqData)
	if err != nil {
		return nil, err
	}

	var resp ads.ReadResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, ads.Error(resp.Result)
	}

	return resp.Data, nil
}

// Write writes data to the ADS device.
func (c *Client) Write(ctx context.Context, indexGroup, indexOffset uint32, data []byte) error {
	req := ads.WriteRequest{
		IndexGroup:  indexGroup,
		IndexOffset: indexOffset,
		Length:      uint32(len(data)),
		Data:        data,
	}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdWrite, reqData)
	if err != nil {
		return err
	}

	var resp ads.WriteResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return err
	}

	if resp.Result != 0 {
		return ads.Error(resp.Result)
	}

	return nil
}

// ReadState reads the ADS and device state.
func (c *Client) ReadState(ctx context.Context) (*DeviceState, error) {
	req := ads.ReadStateRequest{}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdReadState, reqData)
	if err != nil {
		return nil, err
	}

	var resp ads.ReadStateResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, ads.Error(resp.Result)
	}

	return &DeviceState{
		ADSState:    resp.ADSState,
		DeviceState: resp.DeviceState,
	}, nil
}

// WriteControl changes the ADS state of the device.
// This can be used to start, stop, reset the PLC, or perform other state transitions.
// The data parameter is optional and can be nil for most operations.
func (c *Client) WriteControl(ctx context.Context, adsState ads.ADSState, deviceState uint16, data []byte) error {
	req := ads.WriteControlRequest{
		ADSState:    adsState,
		DeviceState: deviceState,
		Length:      uint32(len(data)),
		Data:        data,
	}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdWriteControl, reqData)
	if err != nil {
		return err
	}

	var resp ads.WriteControlResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return err
	}

	if resp.Result != 0 {
		return ads.Error(resp.Result)
	}

	return nil
}

// ReadWrite writes and reads data in a single operation.
func (c *Client) ReadWrite(ctx context.Context, indexGroup, indexOffset, readLength uint32, writeData []byte) ([]byte, error) {
	req := ads.ReadWriteRequest{
		IndexGroup:  indexGroup,
		IndexOffset: indexOffset,
		ReadLength:  readLength,
		WriteLength: uint32(len(writeData)),
		Data:        writeData,
	}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdReadWrite, reqData)
	if err != nil {
		return nil, err
	}

	var resp ads.ReadWriteResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, ads.Error(resp.Result)
	}

	return resp.Data, nil
}
