// Package ams implements AMS (Automation Message Specification) protocol handling.
package ams

import (
	"encoding/binary"
	"fmt"
)

// NetID represents a 6-byte AMS NetID address (e.g., 192.168.1.100.1.1).
// Each byte is stored separately and has no direct relation to IP addresses.
type NetID [6]byte

// String returns the dot-separated string representation of the NetID.
func (n NetID) String() string {
	return fmt.Sprintf("%d.%d.%d.%d.%d.%d", n[0], n[1], n[2], n[3], n[4], n[5])
}

// Port represents a 2-byte AMS port identifier.
type Port uint16

// TCPHeader represents the 6-byte AMS/TCP packet header that precedes the AMS header.
// It contains the length of the following data (AMS Header + ADS Data).
type TCPHeader struct {
	Reserved uint16 // Must be 0
	Length   uint32 // Length of AMS Header + ADS Data in bytes
}

// MarshalBinary encodes the TCPHeader into a 6-byte slice (little-endian).
func (h *TCPHeader) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 6)
	binary.LittleEndian.PutUint16(buf[0:2], h.Reserved)
	binary.LittleEndian.PutUint32(buf[2:6], h.Length)
	return buf, nil
}

// UnmarshalBinary decodes a 6-byte slice into the TCPHeader (little-endian).
func (h *TCPHeader) UnmarshalBinary(data []byte) error {
	if len(data) < 6 {
		return fmt.Errorf("ams: TCP header requires 6 bytes, got %d", len(data))
	}
	h.Reserved = binary.LittleEndian.Uint16(data[0:2])
	h.Length = binary.LittleEndian.Uint32(data[2:6])
	return nil
}

// Header represents the 32-byte AMS header that follows the AMS/TCP header.
// All multi-byte fields are little-endian.
type Header struct {
	TargetNetID NetID  // Destination AMS NetID (6 bytes, offset 0)
	TargetPort  Port   // Destination AMS Port (2 bytes, offset 6)
	SourceNetID NetID  // Source AMS NetID (6 bytes, offset 8)
	SourcePort  Port   // Source AMS Port (2 bytes, offset 14)
	CommandID   uint16 // ADS Command ID (2 bytes, offset 16)
	StateFlags  uint16 // Request/Response and protocol flags (2 bytes, offset 18)
	DataLength  uint32 // Size of ADS data in bytes (4 bytes, offset 20)
	ErrorCode   uint32 // AMS error number (4 bytes, offset 24)
	InvokeID    uint32 // Free usable ID for request/response matching (4 bytes, offset 28)
}

// MarshalBinary encodes the AMS Header into a 32-byte slice (little-endian).
func (h *Header) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 32)

	// Target NetID (6 bytes)
	copy(buf[0:6], h.TargetNetID[:])

	// Target Port (2 bytes)
	binary.LittleEndian.PutUint16(buf[6:8], uint16(h.TargetPort))

	// Source NetID (6 bytes)
	copy(buf[8:14], h.SourceNetID[:])

	// Source Port (2 bytes)
	binary.LittleEndian.PutUint16(buf[14:16], uint16(h.SourcePort))

	// Command ID (2 bytes)
	binary.LittleEndian.PutUint16(buf[16:18], h.CommandID)

	// State Flags (2 bytes)
	binary.LittleEndian.PutUint16(buf[18:20], h.StateFlags)

	// Data Length (4 bytes)
	binary.LittleEndian.PutUint32(buf[20:24], h.DataLength)

	// Error Code (4 bytes)
	binary.LittleEndian.PutUint32(buf[24:28], h.ErrorCode)

	// Invoke ID (4 bytes)
	binary.LittleEndian.PutUint32(buf[28:32], h.InvokeID)

	return buf, nil
}

// UnmarshalBinary decodes a 32-byte slice into the AMS Header (little-endian).
func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) < 32 {
		return fmt.Errorf("ams: header requires 32 bytes, got %d", len(data))
	}

	// Target NetID (6 bytes)
	copy(h.TargetNetID[:], data[0:6])

	// Target Port (2 bytes)
	h.TargetPort = Port(binary.LittleEndian.Uint16(data[6:8]))

	// Source NetID (6 bytes)
	copy(h.SourceNetID[:], data[8:14])

	// Source Port (2 bytes)
	h.SourcePort = Port(binary.LittleEndian.Uint16(data[14:16]))

	// Command ID (2 bytes)
	h.CommandID = binary.LittleEndian.Uint16(data[16:18])

	// State Flags (2 bytes)
	h.StateFlags = binary.LittleEndian.Uint16(data[18:20])

	// Data Length (4 bytes)
	h.DataLength = binary.LittleEndian.Uint32(data[20:24])

	// Error Code (4 bytes)
	h.ErrorCode = binary.LittleEndian.Uint32(data[24:28])

	// Invoke ID (4 bytes)
	h.InvokeID = binary.LittleEndian.Uint32(data[28:32])

	return nil
}

// IsRequest returns true if the StateFlags indicate this is a request packet.
func (h *Header) IsRequest() bool {
	return (h.StateFlags & StateFlagResponse) == 0
}

// IsResponse returns true if the StateFlags indicate this is a response packet.
func (h *Header) IsResponse() bool {
	return (h.StateFlags & StateFlagResponse) != 0
}
