// Package goadstc provides a Go client library for TwinCAT ADS/AMS communication over TCP.
package goadstc

import (
	"context"
	"encoding/binary"
	"fmt"
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
