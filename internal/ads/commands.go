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
	IndexGroupPLCMemory           uint32 = 0x00004020
	IndexGroupPLCMemoryBit        uint32 = 0x00004021
	IndexGroupPhysicalInputs      uint32 = 0x0000F020
	IndexGroupPhysicalInputsBit   uint32 = 0x0000F021
	IndexGroupPhysicalOutputs     uint32 = 0x0000F030
	IndexGroupPhysicalOutputsBit  uint32 = 0x0000F031
	IndexGroupSumCommandRead      uint32 = 0x0000F080
	IndexGroupSumCommandWrite     uint32 = 0x0000F081
	IndexGroupSumCommandReadWrite uint32 = 0x0000F082
)

// TransmissionMode defines how notifications are transmitted.
type TransmissionMode uint32

const (
	TransModeCyclic         TransmissionMode = 3 // Cyclic transmission
	TransModeOnChange       TransmissionMode = 4 // On change transmission
	TransModeCyclicOnChange TransmissionMode = 5 // Cyclic and on change
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

var adsStateNames = map[ADSState]string{
	StateInvalid:    "Invalid",
	StateIdle:       "Idle",
	StateReset:      "Reset",
	StateInit:       "Init",
	StateStart:      "Start",
	StateRun:        "Run",
	StateStop:       "Stop",
	StateSaveConfig: "SaveConfig",
	StateLoadConfig: "LoadConfig",
	StatePowerGood:  "PowerGood",
	StateError:      "Error",
	StateShutdown:   "Shutdown",
	StateSuspend:    "Suspend",
	StateResume:     "Resume",
	StateConfig:     "Config",
	StateReconfig:   "Reconfig",
	StateStop2:      "Stop2",
}

// String returns the string representation of the ADS state.
func (s ADSState) String() string {
	if name, ok := adsStateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", s)
}

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

// AddDeviceNotificationRequest represents an ADS AddDeviceNotification request.
type AddDeviceNotificationRequest struct {
	IndexGroup       uint32
	IndexOffset      uint32
	Length           uint32
	TransmissionMode TransmissionMode
	MaxDelay         uint32 // in milliseconds
	CycleTime        uint32 // in milliseconds
}

func (a *AddDeviceNotificationRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 40)
	binary.LittleEndian.PutUint32(buf[0:4], a.IndexGroup)
	binary.LittleEndian.PutUint32(buf[4:8], a.IndexOffset)
	binary.LittleEndian.PutUint32(buf[8:12], a.Length)
	binary.LittleEndian.PutUint32(buf[12:16], uint32(a.TransmissionMode))
	binary.LittleEndian.PutUint32(buf[16:20], a.MaxDelay)
	binary.LittleEndian.PutUint32(buf[20:24], a.CycleTime)
	// Reserved: 16 bytes (24-39) - already zero
	return buf, nil
}

// AddDeviceNotificationResponse represents an ADS AddDeviceNotification response.
type AddDeviceNotificationResponse struct {
	Result             uint32
	NotificationHandle uint32
}

func (a *AddDeviceNotificationResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("ads: add notification response requires 8 bytes")
	}
	a.Result = binary.LittleEndian.Uint32(data[0:4])
	a.NotificationHandle = binary.LittleEndian.Uint32(data[4:8])
	return nil
}

// DeleteDeviceNotificationRequest represents an ADS DeleteDeviceNotification request.
type DeleteDeviceNotificationRequest struct {
	NotificationHandle uint32
}

func (d *DeleteDeviceNotificationRequest) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf[0:4], d.NotificationHandle)
	return buf, nil
}

// DeleteDeviceNotificationResponse represents an ADS DeleteDeviceNotification response.
type DeleteDeviceNotificationResponse struct {
	Result uint32
}

func (d *DeleteDeviceNotificationResponse) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("ads: delete notification response requires 4 bytes")
	}
	d.Result = binary.LittleEndian.Uint32(data[0:4])
	return nil
}

// NotificationSample represents a single notification sample.
type NotificationSample struct {
	NotificationHandle uint32
	Data               []byte
}

// StampHeader represents a timestamp header with multiple samples.
type StampHeader struct {
	Timestamp uint64 // Windows FILETIME (100ns since 1601-01-01)
	Samples   []NotificationSample
}

// DeviceNotificationRequest represents an ADS DeviceNotification (server push).
type DeviceNotificationRequest struct {
	StampHeaders []StampHeader
}

func (d *DeviceNotificationRequest) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("ads: device notification requires at least 8 bytes")
	}

	length := binary.LittleEndian.Uint32(data[0:4])
	stamps := binary.LittleEndian.Uint32(data[4:8])

	// Length field does not include the first 4 bytes (itself)
	// Expected total: 4 (length field) + length
	expectedTotal := 4 + length
	if uint32(len(data)) < expectedTotal {
		return fmt.Errorf("ads: insufficient data for device notification (expected %d, got %d)", expectedTotal, len(data))
	}

	d.StampHeaders = make([]StampHeader, 0, stamps)
	offset := 8

	for i := uint32(0); i < stamps; i++ {
		if offset+12 > len(data) {
			return fmt.Errorf("ads: insufficient data for stamp header")
		}

		timestamp := binary.LittleEndian.Uint64(data[offset : offset+8])
		sampleCount := binary.LittleEndian.Uint32(data[offset+8 : offset+12])
		offset += 12

		samples := make([]NotificationSample, 0, sampleCount)
		for j := uint32(0); j < sampleCount; j++ {
			if offset+8 > len(data) {
				return fmt.Errorf("ads: insufficient data for notification sample")
			}

			handle := binary.LittleEndian.Uint32(data[offset : offset+4])
			sampleSize := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
			offset += 8

			if offset+int(sampleSize) > len(data) {
				return fmt.Errorf("ads: insufficient data for sample data")
			}

			sampleData := make([]byte, sampleSize)
			copy(sampleData, data[offset:offset+int(sampleSize)])
			offset += int(sampleSize)

			samples = append(samples, NotificationSample{
				NotificationHandle: handle,
				Data:               sampleData,
			})
		}

		d.StampHeaders = append(d.StampHeaders, StampHeader{
			Timestamp: timestamp,
			Samples:   samples,
		})
	}

	return nil
}
