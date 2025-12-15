// Package goadstc provides a Go client library for TwinCAT ADS/AMS communication over TCP.
package goadstc

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
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

	// Use Read command with ADSIGRP_SYM_UPLOAD (0xF00B)
	// Request a large buffer for the symbol table
	readLength := symbolLength
	if readLength < 0xFFFFFF {
		readLength = 0xFFFFFF // Request max to ensure we get everything
	}

	readData, err := c.Read(ctx, 0xF00B, 0, readLength)
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

// ReadSymbol reads data from a PLC symbol by name.
// Automatically loads symbol table on first call.
func (c *Client) ReadSymbol(ctx context.Context, symbolName string) ([]byte, error) {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, err
	}

	symbol, err := c.GetSymbol(symbolName)
	if err != nil {
		return nil, fmt.Errorf("read symbol %q: %w", symbolName, err)
	}

	return c.Read(ctx, symbol.IndexGroup, symbol.IndexOffset, symbol.Size)
}

// WriteSymbol writes data to a PLC symbol by name.
// Automatically loads symbol table on first call.
func (c *Client) WriteSymbol(ctx context.Context, symbolName string, data []byte) error {
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return err
	}

	symbol, err := c.GetSymbol(symbolName)
	if err != nil {
		return fmt.Errorf("write symbol %q: %w", symbolName, err)
	}

	if uint32(len(data)) != symbol.Size {
		return fmt.Errorf("write symbol %q: data size mismatch (expected %d bytes, got %d)",
			symbolName, symbol.Size, len(data))
	}

	return c.Write(ctx, symbol.IndexGroup, symbol.IndexOffset, data)
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

// Subscribe creates a new notification subscription.
// The returned Subscription will deliver notifications via its Notifications() channel.
// Call Close() on the Subscription when done to clean up resources.
func (c *Client) Subscribe(ctx context.Context, opts NotificationOptions) (*Subscription, error) {
	req := ads.AddDeviceNotificationRequest{
		IndexGroup:       opts.IndexGroup,
		IndexOffset:      opts.IndexOffset,
		Length:           opts.Length,
		TransmissionMode: opts.TransmissionMode,
		MaxDelay:         uint32(opts.MaxDelay / time.Millisecond),
		CycleTime:        uint32(opts.CycleTime / time.Millisecond),
	}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdAddDeviceNotification, reqData)
	if err != nil {
		return nil, err
	}

	var resp ads.AddDeviceNotificationResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, ads.Error(resp.Result)
	}

	// Create subscription
	sub := &Subscription{
		handle:  resp.NotificationHandle,
		client:  c,
		notifCh: make(chan Notification, 16),
		closed:  false,
		closeMu: sync.Mutex{},
	}

	// Register subscription
	c.subscriptionsMu.Lock()
	c.subscriptions[sub.handle] = sub
	c.subscriptionsMu.Unlock()

	return sub, nil
}

// unregisterSubscription removes a subscription from the registry.
func (c *Client) unregisterSubscription(handle uint32) {
	c.subscriptionsMu.Lock()
	delete(c.subscriptions, handle)
	c.subscriptionsMu.Unlock()
}

// handleNotification processes incoming notification packets and routes them to subscriptions.
func (c *Client) handleNotification(packet *ams.Packet) {
	var notifReq ads.DeviceNotificationRequest
	if err := notifReq.UnmarshalBinary(packet.Data); err != nil {
		return
	}

	// Process each stamp in the notification
	for _, stamp := range notifReq.StampHeaders {
		// Convert Windows FILETIME to time.Time
		// FILETIME is 100-nanosecond intervals since 1601-01-01 00:00:00 UTC
		const fileTimeEpoch = 116444736000000000 // 100ns intervals between 1601 and 1970
		unixNano := int64(stamp.Timestamp-fileTimeEpoch) * 100
		timestamp := time.Unix(0, unixNano)

		for _, sample := range stamp.Samples {
			c.subscriptionsMu.RLock()
			sub, exists := c.subscriptions[sample.NotificationHandle]
			c.subscriptionsMu.RUnlock()

			if exists {
				sub.notify(sample.Data, timestamp)
			}
		}
	}
}

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

	// Get the symbol
	symbol, err := c.symbolTable.Get(symbolName)
	if err != nil {
		return nil, fmt.Errorf("get symbol %q: %w", symbolName, err)
	}

	if !symbol.Type.IsStruct {
		return nil, fmt.Errorf("%q is not a struct type", symbolName)
	}

	// Read the entire struct data
	structData, err := c.ReadSymbol(ctx, symbolName)
	if err != nil {
		return nil, fmt.Errorf("read struct %q: %w", symbolName, err)
	}

	// Parse the struct based on available type information
	result := make(map[string]interface{})

	// Check if type is registered in the type registry
	var typeInfo symbols.TypeInfo
	var hasTypeInfo bool

	c.typeRegistryMu.RLock()
	typeInfo, hasTypeInfo = c.typeRegistry.Get(symbol.Type.Name)
	c.typeRegistryMu.RUnlock()

	// If not registered, try to fetch from PLC and cache it
	if !hasTypeInfo || len(typeInfo.Fields) == 0 {
		if fetchedTypeInfo, err := c.fetchTypeInfoFromPLC(ctx, symbol.Type.Name); err == nil {
			typeInfo = fetchedTypeInfo
			hasTypeInfo = true
			// Cache it for future use
			c.typeRegistryMu.Lock()
			c.typeRegistry.Register(symbol.Type.Name, typeInfo)
			c.typeRegistryMu.Unlock()
		}
	}

	// Use type info if available
	if hasTypeInfo && len(typeInfo.Fields) > 0 {
		for _, field := range typeInfo.Fields {
			if int(field.Offset)+int(field.Type.Size) > len(structData) {
				continue // Skip fields beyond data bounds
			}
			fieldData := structData[field.Offset : field.Offset+field.Type.Size]
			result[field.Name] = parseFieldValue(fieldData, field.Type)
		}
		return result, nil
	}

	// Fall back to symbol table field information (rarely has detail)
	if len(symbol.Type.Fields) > 0 {
		for _, field := range symbol.Type.Fields {
			if int(field.Offset)+int(field.Type.Size) > len(structData) {
				continue // Skip fields beyond data bounds
			}
			fieldData := structData[field.Offset : field.Offset+field.Type.Size]
			result[field.Name] = parseFieldValue(fieldData, field.Type)
		}
	} else {
		// No detailed field info available
		result["_raw"] = structData
		result["_size"] = len(structData)
		result["_type"] = symbol.Type.Name
		result["_note"] = "Type information not available from PLC. Data type upload may not be supported by this TwinCAT version."
	}

	return result, nil
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

	// Handle nested structs - recursively parse if we have field info
	if typeInfo.IsStruct {
		if len(typeInfo.Fields) > 0 {
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
		return fmt.Sprintf("<struct %s, %d bytes>", typeInfo.Name, len(data))
	}

	// Parse simple types
	switch typeInfo.BaseType {
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
		// Find null terminator
		for i, b := range data {
			if b == 0 {
				return string(data[:i])
			}
		}
		return string(data)
	}

	// Default: return hex string
	return fmt.Sprintf("0x%x", data)
}
