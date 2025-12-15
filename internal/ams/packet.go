package ams

import (
	"fmt"
	"io"
)

// Packet represents a complete AMS packet consisting of TCP header, AMS header, and data.
type Packet struct {
	TCPHeader TCPHeader
	Header    Header
	Data      []byte
}

// NewRequestPacket creates a new request packet with the given parameters.
func NewRequestPacket(targetNetID NetID, targetPort Port, sourceNetID NetID, sourcePort Port, commandID uint16, invokeID uint32, data []byte) *Packet {
	return &Packet{
		TCPHeader: TCPHeader{
			Reserved: 0,
			Length:   32 + uint32(len(data)), // AMS Header (32) + Data
		},
		Header: Header{
			TargetNetID: targetNetID,
			TargetPort:  targetPort,
			SourceNetID: sourceNetID,
			SourcePort:  sourcePort,
			CommandID:   commandID,
			StateFlags:  StateFlagsTCPRequest,
			DataLength:  uint32(len(data)),
			ErrorCode:   0,
			InvokeID:    invokeID,
		},
		Data: data,
	}
}

// MarshalBinary encodes the complete packet (TCP header + AMS header + data).
func (p *Packet) MarshalBinary() ([]byte, error) {
	// Encode TCP header
	tcpBuf, err := p.TCPHeader.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("ams: marshal TCP header: %w", err)
	}

	// Encode AMS header
	amsBuf, err := p.Header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("ams: marshal AMS header: %w", err)
	}

	// Combine all parts
	totalLen := len(tcpBuf) + len(amsBuf) + len(p.Data)
	buf := make([]byte, totalLen)
	offset := 0

	copy(buf[offset:], tcpBuf)
	offset += len(tcpBuf)

	copy(buf[offset:], amsBuf)
	offset += len(amsBuf)

	copy(buf[offset:], p.Data)

	return buf, nil
}

// UnmarshalBinary decodes a complete packet from a byte slice.
func (p *Packet) UnmarshalBinary(data []byte) error {
	if len(data) < 38 { // TCP header (6) + AMS header (32) = 38
		return fmt.Errorf("ams: packet requires at least 38 bytes, got %d", len(data))
	}

	// Decode TCP header
	if err := p.TCPHeader.UnmarshalBinary(data[0:6]); err != nil {
		return fmt.Errorf("ams: unmarshal TCP header: %w", err)
	}

	// Decode AMS header
	if err := p.Header.UnmarshalBinary(data[6:38]); err != nil {
		return fmt.Errorf("ams: unmarshal AMS header: %w", err)
	}

	// Extract data
	expectedLen := 6 + p.TCPHeader.Length
	if uint32(len(data)) < expectedLen {
		return fmt.Errorf("ams: packet data mismatch: expected %d bytes, got %d", expectedLen, len(data))
	}

	dataLen := p.Header.DataLength
	if dataLen > 0 {
		p.Data = make([]byte, dataLen)
		copy(p.Data, data[38:38+dataLen])
	}

	return nil
}

// ReadPacket reads a complete AMS packet from an io.Reader.
// It first reads the TCP header to determine the packet size, then reads the rest.
func ReadPacket(r io.Reader) (*Packet, error) {
	// Read TCP header (6 bytes)
	tcpBuf := make([]byte, 6)
	if _, err := io.ReadFull(r, tcpBuf); err != nil {
		return nil, fmt.Errorf("ams: read TCP header: %w", err)
	}

	var tcpHeader TCPHeader
	if err := tcpHeader.UnmarshalBinary(tcpBuf); err != nil {
		return nil, fmt.Errorf("ams: unmarshal TCP header: %w", err)
	}

	// Read AMS header + data (length from TCP header)
	payloadBuf := make([]byte, tcpHeader.Length)
	if _, err := io.ReadFull(r, payloadBuf); err != nil {
		return nil, fmt.Errorf("ams: read AMS payload: %w", err)
	}

	// Parse AMS header
	var amsHeader Header
	if err := amsHeader.UnmarshalBinary(payloadBuf[0:32]); err != nil {
		return nil, fmt.Errorf("ams: unmarshal AMS header: %w", err)
	}

	// Extract data
	var data []byte
	if amsHeader.DataLength > 0 {
		if uint32(len(payloadBuf)) < 32+amsHeader.DataLength {
			return nil, fmt.Errorf("ams: insufficient data: expected %d bytes, got %d", 32+amsHeader.DataLength, len(payloadBuf))
		}
		data = payloadBuf[32 : 32+amsHeader.DataLength]
	}

	return &Packet{
		TCPHeader: tcpHeader,
		Header:    amsHeader,
		Data:      data,
	}, nil
}

// WritePacket writes a complete AMS packet to an io.Writer.
func WritePacket(w io.Writer, p *Packet) error {
	buf, err := p.MarshalBinary()
	if err != nil {
		return fmt.Errorf("ams: marshal packet: %w", err)
	}

	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("ams: write packet: %w", err)
	}

	return nil
}
