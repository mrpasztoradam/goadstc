# Changelog

## [Unreleased]

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

### Documentation

- Added `OBSERVABILITY.md` with comprehensive observability guide
- Updated README.md with observability features section
- Added inline documentation for all new types and interfaces
