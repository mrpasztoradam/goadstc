// Package ads implements ADS (Automation Device Specification) command handling.
package ads

import (
	"encoding/binary"
	"fmt"
)

type CommandID uint16

const (
	CmdInvalid               CommandID = 0x0000
	CmdReadDeviceInfo        CommandID = 0x0001
	CmdRead                  CommandID = 0x0002
	CmdWrite                 CommandID = 0x0003
	CmdReadState             CommandID = 0x0004
	CmdWriteControl          CommandID = 0x0005
	CmdAddDeviceNotification CommandID = 0x0006
	CmdDelDeviceNotification CommandID = 0x0007
	CmdDeviceNotification    CommandID = 0x0008
	CmdReadWrite             CommandID = 0x0009
)

const (
	IndexGroupPLCMemory              uint32 = 0x00004020
	IndexGroupPLCMemoryBit           uint32 = 0x00004021
	IndexGroupPhysicalInputs         uint32 = 0x0000F020
	IndexGroupPhysicalInputsBit      uint32 = 0x0000F021
	IndexGroupPhysicalOutputs        uint32 = 0x0000F030
	IndexGroupPhysicalOutputsBit     uint32 = 0x0000F031
	IndexGroupSumCommandRead         uint32 = 0x0000F080
	IndexGroupSumCommandWrite        uint32 = 0x0000F081
	IndexGroupSumCommandReadWrite    uint32 = 0x0000F082
)

type ADSState uint16

const (
	StateInvalid    ADSState = 0
	StateIdle       ADSState = 1
	StateReset      ADSState = 2
	StateInit       ADSState = 3
	StateStart      ADSState = 4
	StateRun        ADSState = 5
	StateStop       ADSState = 6
	StateSaveConfig ADSState = 7
	StateLoadConfig ADSState = 8
	StatePowerGood  ADSState = 9
	StateError      ADSState = 10
	StateShutdown   ADSState = 11
	StateSuspend    ADSState = 12
	StateResume     ADSState = 13
	StateConfig     ADSState = 14
	StateReconfig   ADSState = 15
	StateStop2      ADSState = 16
)

type ReadRequest struct {
	IndexGroup  uint32
	IndexOffset uint32
	Length      uint32
}

func (r *ReadRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:4], r.IndexGroup)
	binary.LittleEndian.PutUint32(buf[4:8], r.IndexOffset)
	binary.LittleEndian.PutUint32(buf[8:12], r.Length)
	return buf, nil
}

type ReadResponse struct {
	Result uint32
	Length uint32
	Data   []byte
}

func (r *ReadResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("ads: read response requires at least 8 bytes")
	}
	r.Result = binary.LittleEndian.Uint32(data[0:4])
	r.Length = binary.LittleEndian.Uint32(data[4:8])
	r.Data = make([]byte, r.Length)
	copy(r.Data, data[8:8+r.Length])
	return nil
}

type WriteRequest struct {
	IndexGroup  uint32
	IndexOffset uint32
	Length      uint32
	Data        []byte
}

func (w *WriteRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 12+len(w.Data))
	binary.LittleEndian.PutUint32(buf[0:4], w.IndexGroup)
	binary.LittleEndian.PutUint32(buf[4:8], w.IndexOffset)
	binary.LittleEndian.PutUint32(buf[8:12], w.Length)
	copy(buf[12:], w.Data)
	return buf, nil
}

type WriteResponse struct {
	Result uint32
}

func (w *WriteResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("ads: write response requires 4 bytes")
	}
	w.Result = binary.LittleEndian.Uint32(data[0:4])
	return nil
}

type ReadStateRequest struct{}

func (r *ReadStateRequest) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type ReadStateResponse struct {
	Result      uint32
	ADSState    ADSState
	DeviceState uint16
}

func (r *ReadStateResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("ads: read state response requires 8 bytes")
	}
	r.Result = binary.LittleEndian.Uint32(data[0:4])
	r.ADSState = ADSState(binary.LittleEndian.Uint16(data[4:6]))
	r.DeviceState = binary.LittleEndian.Uint16(data[6:8])
	return nil
}

type ReadDeviceInfoRequest struct{}

func (r *ReadDeviceInfoRequest) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

type ReadDeviceInfoResponse struct {
	Result       uint32
	MajorVersion uint8
	MinorVersion uint8
	VersionBuild uint16
	DeviceName   string
}

func (r *ReadDeviceInfoResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 24 {
		return fmt.Errorf("ads: read device info response requires 24 bytes")
	}
	r.Result = binary.LittleEndian.Uint32(data[0:4])
	r.MajorVersion = data[4]
	r.MinorVersion = data[5]
	r.VersionBuild = binary.LittleEndian.Uint16(data[6:8])
	
	nameBytes := data[8:24]
	nameLen := 0
	for i, b := range nameBytes {
		if b == 0 {
			nameLen = i
			break
		}
	}
	if nameLen == 0 {
		nameLen = 16
	}
	r.DeviceName = string(nameBytes[:nameLen])
	return nil
}

type ReadWriteRequest struct {
	IndexGroup  uint32
	IndexOffset uint32
	ReadLength  uint32
	WriteLength uint32
	Data        []byte
}

func (r *ReadWriteRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 16+len(r.Data))
	binary.LittleEndian.PutUint32(buf[0:4], r.IndexGroup)
	binary.LittleEndian.PutUint32(buf[4:8], r.IndexOffset)
	binary.LittleEndian.PutUint32(buf[8:12], r.ReadLength)
	binary.LittleEndian.PutUint32(buf[12:16], r.WriteLength)
	copy(buf[16:], r.Data)
	return buf, nil
}

type ReadWriteResponse struct {
	Result uint32
	Length uint32
	Data   []byte
}

func (r *ReadWriteResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("ads: read/write response requires at least 8 bytes")
	}
	r.Result = binary.LittleEndian.Uint32(data[0:4])
	r.Length = binary.LittleEndian.Uint32(data[4:8])
	r.Data = make([]byte, r.Length)
	copy(r.Data, data[8:8+r.Length])
	return nil
}

type WriteControlRequest struct {
	ADSState    ADSState
	DeviceState uint16
	Length      uint32
	Data        []byte
}

func (w *WriteControlRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8+len(w.Data))
	binary.LittleEndian.PutUint16(buf[0:2], uint16(w.ADSState))
	binary.LittleEndian.PutUint16(buf[2:4], w.DeviceState)
	binary.LittleEndian.PutUint32(buf[4:8], w.Length)
	copy(buf[8:], w.Data)
	return buf, nil
}

type WriteControlResponse struct {
	Result uint32
}

func (w *WriteControlResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("ads: write control response requires 4 bytes")
	}
	w.Result = binary.LittleEndian.Uint32(data[0:4])
	return nil
}
