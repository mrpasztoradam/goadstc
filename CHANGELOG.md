# Changelog

## [Unreleased]

### Added

- **Automatic Type Detection and Parsing**
  - `ReadSymbolValue()` - Reads any symbol and automatically parses to appropriate Go type
  - Supports all basic types (INT, REAL, BOOL, STRING, etc.)
  - Automatically parses structs into `map[string]interface{}` with all fields
  - Recursively parses nested structs and arrays
  - Fetches type information from PLC using data type upload protocol
  
- **Automatic Type Encoding for Writes**
  - `WriteSymbolValue()` - Automatically encodes Go values based on symbol type
  - Supports all Go primitive types and common types (`time.Duration`, `time.Time`)
  - Validates types against PLC symbol type
  - Eliminates need for manual byte encoding

- **Enhanced Data Type Upload**
  - Improved `UploadDataTypeTable()` for fetching all type definitions from PLC
  - Automatic caching of type information in `TypeRegistry`
  - Recursive type information fetching for nested structs
  - Support for complex struct hierarchies

- **Batch Reading**
  - `ReadMultipleSymbolValues()` - Read multiple symbols in one call
  - Returns map with symbol names as keys
  - API ready for future SumCommand optimization (0xF080)

- **New Example**
  - Complete auto-type detection example in `examples/auto-type/`
  - Demonstrates automatic reading, writing, and batch operations
  - Shows struct and array handling

### Changed

- Deprecated redundant struct field methods in favor of dot notation
  - `ReadStructFieldInt16()` → Use `ReadInt16(ctx, "MAIN.struct.field")`
  - All `ReadStructField*` and `WriteStructField*` methods marked deprecated
  - Backward compatible - methods still available but discouraged

- Deprecated `ReadStructAsMap()` in favor of `ReadSymbolValue()`
  - `ReadSymbolValue()` provides same functionality with better ergonomics
  - No manual type registration needed
  - Backward compatible - method still available

### Improved

- Enhanced type information handling in symbol parser
- Better error messages for type detection failures
- Improved struct field extraction and parsing
- More efficient type registry lookups with better caching

### Documentation

- Updated README.md with automatic type detection examples
- Reorganized API documentation for clarity
- Added Quick Start section with automatic type detection
- Enhanced feature list highlighting new capabilities

## [0.2.1] - 2025-12-16

### Improved

- Further reduced cyclomatic complexity in ADSState.String() method
  - Replaced switch statement with map lookup (18→2 complexity)
  - Improved performance and maintainability

## [0.2.0] - 2025-12-16

### Added

- Library versioning system

  - Version constants (Major, Minor, Patch)
  - `Version()` function returning semantic version string
  - `GetBuildInfo()` for detailed build and VCS information
  - Automatic git commit/tag extraction when built with Go 1.18+
  - version_test.go with comprehensive tests

- Structured logging support using Go's standard `log/slog` package
  - JSON-based logging for easy parsing and analysis
  - Configurable log levels (Debug, Info, Warn, Error)
  - Custom logger interface for integration with existing logging systems
  - No-op logger by default for zero overhead
- Error classification system
  - Automatic categorization into: Network, Protocol, ADS, Validation, Configuration, Timeout, State
  - `ClassifiedError` type with detailed error context
  - Retryability detection for automatic error recovery
  - Enhanced error messages with operation context
- Metrics collection interface
  - Connection metrics (attempts, successes, failures, reconnections)
  - Operation metrics (counts, durations, error rates)
  - Data transfer metrics (bytes sent/received)
  - Notification metrics (received, dropped, active subscriptions)
  - Health check metrics (started, success, failure counts)
  - Error metrics by category and operation
- In-memory metrics collector (`InMemoryMetrics`)
  - Thread-safe metrics accumulation
  - Snapshot support for point-in-time reporting
  - Suitable for testing and debugging
- Observability integration points
  - `WithLogger()` option for client configuration
  - `WithMetrics()` option for client configuration
  - Metrics interface for custom backend integration (Prometheus, StatsD, etc.)
- Example implementation
  - Complete observability example in `examples/observability/`
  - Demonstrates logging, metrics, and error classification

### Changed

- Enhanced error messages throughout the codebase
- All client operations now log detailed information
- Connection lifecycle events now logged with structured fields
- Notification handling includes logging and metrics

### Improved

- Code quality improvements
  - Reduced cyclomatic complexity across codebase
  - Refactored `parseFieldValue()` from complexity 32 to 3 in client_structs.go
  - Refactored `ReadStructAsMap()` from complexity 18 to 2 in client_structs.go
  - Refactored `resolveArraySymbol()` from complexity 17 to 2 in client.go
  - Refactored examples (symbols, arrays, structs, comprehensive) to reduce complexity
  - Extracted helper functions for better code organization
- Added `String()` method to `ADSState` type for standard string representation
  - Implements `Stringer` interface
  - Centralized state-to-string conversion in core library
  - Available for all logging and error messages
- Fixed ineffectual assignments in client.go reconnection logic and context handling

### Documentation

- Added `OBSERVABILITY.md` with comprehensive observability guide
- Updated README.md with observability features section
- Added inline documentation for all new types and interfaces
