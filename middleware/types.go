package middleware

import "time"

// SymbolValueResponse represents a single symbol read response
type SymbolValueResponse struct {
	Success bool        `json:"success"`
	Symbol  string      `json:"symbol"`
	Value   interface{} `json:"value"`
	Type    string      `json:"type,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BatchReadRequest represents a request to read multiple symbols
type BatchReadRequest struct {
	Symbols []string `json:"symbols" example:"MAIN.temperature,MAIN.counter"`
}

// BatchReadResponse represents a batch read response
type BatchReadResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Errors  map[string]string      `json:"errors,omitempty"`
}

// WriteSymbolRequest represents a single symbol write request
type WriteSymbolRequest struct {
	Value interface{} `json:"value" example:"25.5"`
}

// WriteSymbolResponse represents a single symbol write response
type WriteSymbolResponse struct {
	Success bool   `json:"success"`
	Symbol  string `json:"symbol"`
	Error   string `json:"error,omitempty"`
}

// BatchWriteRequest represents a request to write multiple symbols
type BatchWriteRequest struct {
	Writes map[string]interface{} `json:"writes" example:"{'MAIN.temperature': 25.5, 'MAIN.enabled': true}"`
}

// BatchWriteResponse represents a batch write response
type BatchWriteResponse struct {
	Success bool              `json:"success"`
	Results map[string]bool   `json:"results"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// WriteStructFieldsRequest represents a request to write struct fields
type WriteStructFieldsRequest struct {
	Fields map[string]interface{} `json:"fields" example:"{'temperature': 25.5, 'enabled': true}"`
}

// WriteStructFieldsResponse represents a struct field write response
type WriteStructFieldsResponse struct {
	Success       bool   `json:"success"`
	FieldsWritten int    `json:"fields_written"`
	Error         string `json:"error,omitempty"`
}

// SymbolInfo represents metadata about a symbol
type SymbolInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        uint32 `json:"size"`
	IndexGroup  uint32 `json:"index_group"`
	IndexOffset uint32 `json:"index_offset"`
	Comment     string `json:"comment,omitempty"`
}

// SymbolTableResponse represents the symbol table response
type SymbolTableResponse struct {
	Success bool         `json:"success"`
	Count   int          `json:"count"`
	Symbols []SymbolInfo `json:"symbols"`
	Error   string       `json:"error,omitempty"`
}

// SubscriptionRequest represents a subscription creation request
type SubscriptionRequest struct {
	Symbol           string `json:"symbol" example:"MAIN.temperature"`
	TransmissionMode string `json:"mode" example:"onchange"`
	CycleTimeMs      int    `json:"cycle_time_ms" example:"100"`
	MaxDelayMs       int    `json:"max_delay_ms" example:"500"`
}

// SubscriptionResponse represents a subscription creation response
type SubscriptionResponse struct {
	Success        bool   `json:"success"`
	SubscriptionID string `json:"subscription_id"`
	Symbol         string `json:"symbol"`
	WebSocketURL   string `json:"websocket_url"`
	Error          string `json:"error,omitempty"`
}

// SubscriptionUpdate represents a subscription value update
type SubscriptionUpdate struct {
	Type           string      `json:"type"`
	SubscriptionID string      `json:"subscription_id"`
	Symbol         string      `json:"symbol"`
	Value          interface{} `json:"value"`
	Timestamp      string      `json:"timestamp"`
}

// SubscriptionError represents a subscription error
type SubscriptionError struct {
	Type           string `json:"type"`
	SubscriptionID string `json:"subscription_id"`
	Error          string `json:"error"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Connected bool      `json:"connected"`
	Timestamp time.Time `json:"timestamp"`
}

// InfoResponse represents PLC connection info
type InfoResponse struct {
	Target       string `json:"target"`
	AMSNetID     string `json:"ams_net_id"`
	SourceNetID  string `json:"source_net_id"`
	AMSPort      uint16 `json:"ams_port"`
	Connected    bool   `json:"connected"`
	SymbolCount  int    `json:"symbol_count"`
	ServerUptime string `json:"server_uptime"`
}

// VersionResponse represents runtime version information
type VersionResponse struct {
	Success      bool   `json:"success"`
	Name         string `json:"name"`
	MajorVersion uint8  `json:"major_version"`
	MinorVersion uint8  `json:"minor_version"`
	VersionBuild uint16 `json:"version_build"`
	Version      string `json:"version"`
	Error        string `json:"error,omitempty"`
}

// StateResponse represents PLC state information
type StateResponse struct {
	Success      bool   `json:"success"`
	ADSState     uint16 `json:"ads_state"`
	ADSStateName string `json:"ads_state_name"`
	DeviceState  uint16 `json:"device_state"`
	Error        string `json:"error,omitempty"`
}

// ControlRequest represents a PLC control operation request
type ControlRequest struct {
	Command string `json:"command"` // "start", "stop", "reset"
}

// ControlResponse represents the result of a control operation
type ControlResponse struct {
	Success bool   `json:"success"`
	Command string `json:"command"`
	Error   string `json:"error,omitempty"`
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}
