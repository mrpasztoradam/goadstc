// Package goadstc provides a Go client library for TwinCAT ADS/AMS communication over TCP.
package goadstc

import (
	"context"
	"fmt"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/ams"
	"github.com/mrpasztoradam/goadstc/internal/transport"
)

// Client represents an ADS client connection.
type Client struct {
	conn        *transport.Conn
	targetNetID ams.NetID
	targetPort  ams.Port
	sourceNetID ams.NetID
	sourcePort  ams.Port
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

	return &Client{
		conn:        conn,
		targetNetID: cfg.targetNetID,
		targetPort:  cfg.targetPort,
		sourceNetID: cfg.sourceNetID,
		sourcePort:  cfg.sourcePort,
	}, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
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
