// Package ads implements symbol resolution commands for TwinCAT 3.
package ads

import (
	"encoding/binary"
	"fmt"
)

// Symbol-related index groups as per ADS specification.
const (
	IndexGroupSymbolHandleByName  uint32 = 0xF003 // Get symbol handle by name
	IndexGroupReleaseSymbolHandle uint32 = 0xF006 // Release symbol handle
	IndexGroupSymbolValueByName   uint32 = 0xF005 // Read/write symbol by name directly
	IndexGroupSymbolInfoByName    uint32 = 0xF007 // Get symbol info by name
	IndexGroupSymbolVersion       uint32 = 0xF008 // Get symbol version
	IndexGroupSymbolUploadInfo    uint32 = 0xF00B // Get upload info (symbol count)
	IndexGroupSymbolUpload        uint32 = 0xF00C // Upload symbol table
	IndexGroupSymbolUploadInfo2   uint32 = 0xF00E // Extended upload info (TC3)
	IndexGroupSymbolUpload2       uint32 = 0xF00F // Extended symbol upload (TC3)
)

// GetSymbolHandleByNameRequest retrieves a handle for a symbol name.
// IndexGroup: 0xF003, IndexOffset: 0x00000000
type GetSymbolHandleByNameRequest struct {
	SymbolName string
}

func (r *GetSymbolHandleByNameRequest) MarshalBinary() ([]byte, error) {
	// Symbol name as null-terminated string
	nameBytes := []byte(r.SymbolName)
	buf := make([]byte, len(nameBytes)+1) // +1 for null terminator
	copy(buf, nameBytes)
	buf[len(nameBytes)] = 0
	return buf, nil
}

// GetSymbolHandleByNameResponse contains the symbol handle.
type GetSymbolHandleByNameResponse struct {
	Handle uint32
}

func (r *GetSymbolHandleByNameResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("ads: symbol handle response requires 4 bytes, got %d", len(data))
	}
	r.Handle = binary.LittleEndian.Uint32(data[0:4])
	return nil
}

// ReleaseSymbolHandleRequest releases a symbol handle.
// IndexGroup: 0xF006, IndexOffset: 0x00000000
type ReleaseSymbolHandleRequest struct {
	Handle uint32
}

func (r *ReleaseSymbolHandleRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf[0:4], r.Handle)
	return buf, nil
}

// ReleaseSymbolHandleResponse is empty (uses standard Result in ReadWrite response).
type ReleaseSymbolHandleResponse struct{}

func (r *ReleaseSymbolHandleResponse) UnmarshalBinary(data []byte) error {
	return nil
}

// SymbolUploadInfoRequest gets information about the symbol table.
// IndexGroup: 0xF00B, IndexOffset: 0x00000000
type SymbolUploadInfoRequest struct{}

func (r *SymbolUploadInfoRequest) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

// SymbolUploadInfoResponse contains symbol table metadata.
type SymbolUploadInfoResponse struct {
	SymbolCount  uint32 // Number of symbols
	SymbolLength uint32 // Total size of symbol data in bytes
}

func (r *SymbolUploadInfoResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("ads: symbol upload info response requires at least 8 bytes, got %d", len(data))
	}
	r.SymbolCount = binary.LittleEndian.Uint32(data[0:4])
	r.SymbolLength = binary.LittleEndian.Uint32(data[4:8])
	// Additional fields may exist in extended response but are optional
	return nil
}

// SymbolUploadRequest requests the complete symbol table.
// IndexGroup: 0xF00C, IndexOffset: 0x00000000
type SymbolUploadRequest struct{}

func (r *SymbolUploadRequest) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

// SymbolUploadResponse contains the raw symbol table data.
// The data format is complex and requires parsing (see parser.go).
type SymbolUploadResponse struct {
	Data []byte
}

func (r *SymbolUploadResponse) UnmarshalBinary(data []byte) error {
	r.Data = make([]byte, len(data))
	copy(r.Data, data)
	return nil
}

// SymbolInfoByNameRequest gets detailed info about a symbol.
// IndexGroup: 0xF007, IndexOffset: 0x00000000
type SymbolInfoByNameRequest struct {
	SymbolName string
}

func (r *SymbolInfoByNameRequest) MarshalBinary() ([]byte, error) {
	nameBytes := []byte(r.SymbolName)
	buf := make([]byte, len(nameBytes)+1)
	copy(buf, nameBytes)
	buf[len(nameBytes)] = 0
	return buf, nil
}

// SymbolEntry represents a parsed symbol from the upload data.
type SymbolEntry struct {
	EntryLength   uint32
	IndexGroup    uint32
	IndexOffset   uint32
	Size          uint32
	DataType      uint32
	Flags         uint32
	NameLength    uint16
	TypeLength    uint16
	CommentLength uint16
	Name          string
	Type          string
	Comment       string
}

// Symbol flags
const (
	SymbolFlagPersistent       uint32 = 0x00000001
	SymbolFlagBitValue         uint32 = 0x00000002
	SymbolFlagRemanent         uint32 = 0x00000008
	SymbolFlagTComInterfacePtr uint32 = 0x00000010
	SymbolFlagTypeGUID         uint32 = 0x00000020
	SymbolFlagAttributes       uint32 = 0x00001000
	SymbolFlagStatic           uint32 = 0x00004000
)
