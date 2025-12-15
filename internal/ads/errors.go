package ads

import "fmt"

type Error uint32

const (
	ErrNoError                           Error = 0x0000
	ErrInternal                          Error = 0x0001
	ErrTargetPortNotFound                Error = 0x0006
	ErrTargetMachineNotFound             Error = 0x0007
	ErrDeviceInvalidIndexGroup           Error = 0x0702
	ErrDeviceInvalidIndexOffset          Error = 0x0703
)

func (e Error) Error() string {
	switch e {
	case ErrNoError:
		return "no error"
	case ErrInternal:
		return "internal error"
	case ErrTargetPortNotFound:
		return "target port not found"
	case ErrTargetMachineNotFound:
		return "target machine not found"
	case ErrDeviceInvalidIndexGroup:
		return "invalid index group"
	case ErrDeviceInvalidIndexOffset:
		return "invalid index offset"
	default:
		return fmt.Sprintf("ADS error 0x%04X", uint32(e))
	}
}

func (e Error) IsError() bool {
	return e != ErrNoError
}
