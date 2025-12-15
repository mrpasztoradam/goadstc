package ams

// State flag bits for the StateFlags field in AMS Header.
const (
	// StateFlagResponse indicates a response packet (bit 0).
	// 0 = Request, 1 = Response
	StateFlagResponse uint16 = 0x0001

	// StateFlagADS must be set for ADS commands (bit 2).
	StateFlagADS uint16 = 0x0004

	// StateFlagUDP indicates UDP protocol (bit 7).
	// 0 = TCP, 1 = UDP
	StateFlagUDP uint16 = 0x0080
)

// Predefined state flag combinations for common use cases.
const (
	// StateFlagsTCPRequest represents a TCP request (0x0004).
	StateFlagsTCPRequest = StateFlagADS

	// StateFlagsTCPResponse represents a TCP response (0x0005).
	StateFlagsTCPResponse = StateFlagADS | StateFlagResponse

	// StateFlagsUDPRequest represents a UDP request (0x0084).
	StateFlagsUDPRequest = StateFlagADS | StateFlagUDP

	// StateFlagsUDPResponse represents a UDP response (0x0085).
	StateFlagsUDPResponse = StateFlagADS | StateFlagUDP | StateFlagResponse
)

// Common AMS port numbers used by TwinCAT runtime.
const (
	PortLogger        Port = 100   // Logger
	PortEventLogger   Port = 110   // EventLogger
	PortRouter        Port = 1     // AMS Router
	PortSystemService Port = 10000 // System Service
	PortPLCRuntime1   Port = 851   // First PLC runtime
	PortPLCRuntime2   Port = 852   // Second PLC runtime
	PortPLCRuntime3   Port = 853   // Third PLC runtime
	PortPLCRuntime4   Port = 854   // Fourth PLC runtime
)
