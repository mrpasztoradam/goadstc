package middleware

import (
	"encoding/json"
	"net/http"
)

// Error codes
const (
	ErrCodeSymbolNotFound        = "SYMBOL_NOT_FOUND"
	ErrCodeInvalidRequest        = "INVALID_REQUEST"
	ErrCodeTypeMismatch          = "TYPE_MISMATCH"
	ErrCodeWriteFailed           = "WRITE_FAILED"
	ErrCodeSubscriptionLimit     = "SUBSCRIPTION_LIMIT_REACHED"
	ErrCodePLCConnectionError    = "PLC_CONNECTION_ERROR"
	ErrCodeInternalError         = "INTERNAL_ERROR"
	ErrCodeSymbolTableLoadFailed = "SYMBOL_TABLE_LOAD_FAILED"
	ErrCodeUnauthorized          = "UNAUTHORIZED"
	ErrCodeBatchSizeExceeded     = "BATCH_SIZE_EXCEEDED"
)

// HTTPError represents an HTTP error with status code and error response
type HTTPError struct {
	StatusCode int
	Response   ErrorResponse
}

// Error implements the error interface
func (e HTTPError) Error() string {
	return e.Response.Error.Message
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, code, message string, details map[string]interface{}) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Response: ErrorResponse{
			Error: ErrorDetail{
				Code:    code,
				Message: message,
				Details: details,
			},
		},
	}
}

// NewSymbolNotFoundError creates a symbol not found error
func NewSymbolNotFoundError(symbol string) *HTTPError {
	return NewHTTPError(
		http.StatusNotFound,
		ErrCodeSymbolNotFound,
		"Symbol not found in PLC",
		map[string]interface{}{"symbol": symbol},
	)
}

// NewInvalidRequestError creates an invalid request error
func NewInvalidRequestError(message string) *HTTPError {
	return NewHTTPError(
		http.StatusBadRequest,
		ErrCodeInvalidRequest,
		message,
		nil,
	)
}

// NewTypeMismatchError creates a type mismatch error
func NewTypeMismatchError(symbol string, expected, got string) *HTTPError {
	return NewHTTPError(
		http.StatusBadRequest,
		ErrCodeTypeMismatch,
		"Type mismatch when writing symbol",
		map[string]interface{}{
			"symbol":   symbol,
			"expected": expected,
			"got":      got,
		},
	)
}

// NewWriteFailedError creates a write failed error
func NewWriteFailedError(symbol, reason string) *HTTPError {
	return NewHTTPError(
		http.StatusInternalServerError,
		ErrCodeWriteFailed,
		"Failed to write symbol value",
		map[string]interface{}{
			"symbol": symbol,
			"reason": reason,
		},
	)
}

// NewPLCConnectionError creates a PLC connection error
func NewPLCConnectionError(message string) *HTTPError {
	return NewHTTPError(
		http.StatusServiceUnavailable,
		ErrCodePLCConnectionError,
		message,
		nil,
	)
}

// NewInternalError creates an internal error
func NewInternalError(message string) *HTTPError {
	return NewHTTPError(
		http.StatusInternalServerError,
		ErrCodeInternalError,
		message,
		nil,
	)
}

// NewBatchSizeExceededError creates a batch size exceeded error
func NewBatchSizeExceededError(requested, max int) *HTTPError {
	return NewHTTPError(
		http.StatusBadRequest,
		ErrCodeBatchSizeExceeded,
		"Batch size exceeds maximum allowed",
		map[string]interface{}{
			"requested": requested,
			"maximum":   max,
		},
	)
}

// WriteError writes an error response to the HTTP response writer
func WriteError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	var ok bool

	if httpErr, ok = err.(*HTTPError); !ok {
		// Convert regular errors to internal errors
		httpErr = NewInternalError(err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.StatusCode)
	json.NewEncoder(w).Encode(httpErr.Response)
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}
