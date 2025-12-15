package goadstc

import (
	"errors"
	"fmt"

	"github.com/mrpasztoradam/goadstc/internal/ads"
)

// ErrorCategory represents the type of error for better error handling.
type ErrorCategory int

const (
	// ErrorCategoryUnknown represents an unclassified error.
	ErrorCategoryUnknown ErrorCategory = iota

	// ErrorCategoryNetwork represents network-level errors (connection, timeout, etc.).
	ErrorCategoryNetwork

	// ErrorCategoryProtocol represents AMS/ADS protocol errors.
	ErrorCategoryProtocol

	// ErrorCategoryADS represents ADS device errors returned by the PLC.
	ErrorCategoryADS

	// ErrorCategoryValidation represents input validation errors.
	ErrorCategoryValidation

	// ErrorCategoryConfiguration represents configuration errors.
	ErrorCategoryConfiguration

	// ErrorCategoryTimeout represents timeout errors.
	ErrorCategoryTimeout

	// ErrorCategoryState represents state-related errors (e.g., client closed).
	ErrorCategoryState
)

func (c ErrorCategory) String() string {
	switch c {
	case ErrorCategoryNetwork:
		return "network"
	case ErrorCategoryProtocol:
		return "protocol"
	case ErrorCategoryADS:
		return "ads"
	case ErrorCategoryValidation:
		return "validation"
	case ErrorCategoryConfiguration:
		return "configuration"
	case ErrorCategoryTimeout:
		return "timeout"
	case ErrorCategoryState:
		return "state"
	default:
		return "unknown"
	}
}

// ClassifiedError wraps an error with additional classification metadata.
type ClassifiedError struct {
	Category    ErrorCategory
	Operation   string // The operation that failed (e.g., "read", "write", "connect")
	Err         error
	Retryable   bool // Whether the operation can be retried
	ADSError    *ads.Error
	SymbolName  string  // Optional: the symbol name if relevant
	IndexGroup  *uint32 // Optional: index group if relevant
	IndexOffset *uint32 // Optional: index offset if relevant
}

func (e *ClassifiedError) Error() string {
	if e.SymbolName != "" {
		return fmt.Sprintf("%s operation failed for symbol %q: %v", e.Operation, e.SymbolName, e.Err)
	}
	return fmt.Sprintf("%s operation failed: %v", e.Operation, e.Err)
}

func (e *ClassifiedError) Unwrap() error {
	return e.Err
}

// IsRetryable returns whether the error indicates a retryable condition.
func (e *ClassifiedError) IsRetryable() bool {
	return e.Retryable
}

// ClassifyError attempts to classify an error into a category.
func ClassifyError(err error, operation string) *ClassifiedError {
	if err == nil {
		return nil
	}

	ce := &ClassifiedError{
		Category:  ErrorCategoryUnknown,
		Operation: operation,
		Err:       err,
		Retryable: false,
	}

	// Check for ADS errors
	var adsErr ads.Error
	if errors.As(err, &adsErr) {
		ce.Category = ErrorCategoryADS
		ce.ADSError = &adsErr
		ce.Retryable = isRetryableADSError(adsErr)
		return ce
	}

	// Check error message for common patterns
	errMsg := err.Error()

	// Network errors
	if errors.Is(err, errors.New("connection closed")) ||
		errors.Is(err, errors.New("connection failed")) ||
		containsAny(errMsg, "connection refused", "connection reset", "broken pipe",
			"network is unreachable", "no route to host", "i/o timeout") {
		ce.Category = ErrorCategoryNetwork
		ce.Retryable = true
		return ce
	}

	// Timeout errors
	if containsAny(errMsg, "timeout", "deadline exceeded", "context deadline exceeded") {
		ce.Category = ErrorCategoryTimeout
		ce.Retryable = true
		return ce
	}

	// State errors
	if containsAny(errMsg, "client closed", "client not connected", "connection closed") {
		ce.Category = ErrorCategoryState
		ce.Retryable = false
		return ce
	}

	// Validation errors
	if containsAny(errMsg, "invalid", "empty", "cannot be empty", "must be positive") {
		ce.Category = ErrorCategoryValidation
		ce.Retryable = false
		return ce
	}

	// Protocol errors
	if containsAny(errMsg, "protocol", "packet", "marshal", "unmarshal", "parse") {
		ce.Category = ErrorCategoryProtocol
		ce.Retryable = false
		return ce
	}

	return ce
}

func isRetryableADSError(err ads.Error) bool {
	switch err {
	case ads.ErrTargetPortNotFound, ads.ErrTargetMachineNotFound:
		return true
	default:
		return false
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexString(s, substr) >= 0)
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Common error constructors with classification

// NewNetworkError creates a classified network error.
func NewNetworkError(operation string, err error) error {
	return &ClassifiedError{
		Category:  ErrorCategoryNetwork,
		Operation: operation,
		Err:       err,
		Retryable: true,
	}
}

// NewValidationError creates a classified validation error.
func NewValidationError(operation, message string) error {
	return &ClassifiedError{
		Category:  ErrorCategoryValidation,
		Operation: operation,
		Err:       errors.New(message),
		Retryable: false,
	}
}

// NewADSError creates a classified ADS error.
func NewADSError(operation string, adsErr ads.Error) error {
	return &ClassifiedError{
		Category:  ErrorCategoryADS,
		Operation: operation,
		Err:       adsErr,
		ADSError:  &adsErr,
		Retryable: isRetryableADSError(adsErr),
	}
}

// NewStateError creates a classified state error.
func NewStateError(operation, message string) error {
	return &ClassifiedError{
		Category:  ErrorCategoryState,
		Operation: operation,
		Err:       errors.New(message),
		Retryable: false,
	}
}
