# Observability Features

This document describes the structured logging, error classification, and metrics features available in goadstc.

## Overview

goadstc provides comprehensive observability features to help monitor and debug ADS client applications:

- **Structured Logging**: JSON-based logging using Go's standard `log/slog` package
- **Error Classification**: Automatic categorization of errors for better error handling
- **Metrics Collection**: Track operations, performance, and connection health

## Structured Logging

### Using the Default Logger

Create a client with structured logging enabled:

```go
import (
    "log/slog"
    "os"
    "github.com/mrpasztoradam/goadstc"
)

// Create a JSON logger
handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
logger := goadstc.NewSlogLogger(slog.New(handler))

// Create client with logger
client, err := goadstc.New(
    goadstc.WithTarget("192.168.1.10:48898"),
    goadstc.WithAMSNetID(netID),
    goadstc.WithLogger(logger),
)
```

### Custom Logger Implementation

You can implement your own logger by implementing the `Logger` interface:

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    With(args ...any) Logger
}
```

### Log Levels and Output

The client logs various events at different levels:

- **Debug**: Detailed operation traces (reads, writes, notifications)
- **Info**: High-level operations (connection, reconnection, subscriptions)
- **Warn**: Recoverable issues (health check failures, dropped notifications)
- **Error**: Operation failures with detailed error context

Example log output:

```json
{"time":"2025-12-15T10:30:45.123Z","level":"INFO","msg":"creating new ADS client","target":"192.168.1.10:48898","targetNetID":"127.0.0.1.1.1","targetPort":851,"autoReconnect":true}
{"time":"2025-12-15T10:30:45.234Z","level":"INFO","msg":"connecting to ADS target","address":"192.168.1.10:48898"}
{"time":"2025-12-15T10:30:45.345Z","level":"INFO","msg":"connected successfully"}
{"time":"2025-12-15T10:30:45.456Z","level":"DEBUG","msg":"reading data","indexGroup":61445,"indexOffset":0,"length":100}
{"time":"2025-12-15T10:30:45.567Z","level":"DEBUG","msg":"read completed","bytes":100,"duration":"111ms"}
```

## Error Classification

All errors returned by the client are automatically classified into categories for better error handling and monitoring.

### Error Categories

```go
const (
    ErrorCategoryUnknown        // Unclassified errors
    ErrorCategoryNetwork        // Network-level errors (connection, timeout)
    ErrorCategoryProtocol       // AMS/ADS protocol errors
    ErrorCategoryADS            // ADS device errors from PLC
    ErrorCategoryValidation     // Input validation errors
    ErrorCategoryConfiguration  // Configuration errors
    ErrorCategoryTimeout        // Timeout errors
    ErrorCategoryState          // State-related errors
)
```

### Using Classified Errors

```go
data, err := client.ReadSymbolByName(ctx, "MAIN.Counter")
if err != nil {
    if ce, ok := err.(*goadstc.ClassifiedError); ok {
        fmt.Printf("Error category: %s\n", ce.Category)
        fmt.Printf("Operation: %s\n", ce.Operation)
        fmt.Printf("Retryable: %v\n", ce.IsRetryable())

        if ce.ADSError != nil {
            fmt.Printf("ADS error code: 0x%04X\n", *ce.ADSError)
        }

        if ce.SymbolName != "" {
            fmt.Printf("Symbol: %s\n", ce.SymbolName)
        }
    }
}
```

### Retryable Errors

The client automatically determines if an error is retryable:

```go
ce := goadstc.ClassifyError(err, "read")
if ce.IsRetryable() {
    // Safe to retry the operation
}
```

## Metrics Collection

### Using In-Memory Metrics

```go
// Create metrics collector
metrics := goadstc.NewInMemoryMetrics()

// Create client with metrics
client, err := goadstc.New(
    goadstc.WithTarget("192.168.1.10:48898"),
    goadstc.WithAMSNetID(netID),
    goadstc.WithMetrics(metrics),
)

// Get metrics snapshot
snapshot := metrics.Snapshot()
fmt.Printf("Connection attempts: %d\n", snapshot.ConnectionAttempts)
fmt.Printf("Operations completed: %d\n", snapshot.OperationCounts["read"])
fmt.Printf("Bytes sent: %d\n", snapshot.BytesSent)
```

### Available Metrics

**Connection Metrics:**

- Connection attempts, successes, failures
- Active connection state
- Reconnection count

**Operation Metrics:**

- Per-operation counts (read, write, subscribe, etc.)
- Operation durations (in-memory collector only)
- Error counts per operation

**Data Transfer:**

- Bytes sent
- Bytes received

**Notifications:**

- Notifications received
- Notifications dropped
- Active subscription count

**Error Metrics:**

- Errors by category
- Errors by operation

**Health Checks:**

- Health checks started
- Health check successes/failures

### Custom Metrics Implementation

Implement the `Metrics` interface to integrate with your monitoring system:

```go
type Metrics interface {
    ConnectionAttempts()
    ConnectionSuccesses()
    ConnectionFailures()
    ConnectionActive(active bool)
    Reconnections()

    OperationStarted(operation string)
    OperationCompleted(operation string, duration time.Duration, err error)

    BytesSent(bytes int64)
    BytesReceived(bytes int64)

    NotificationReceived()
    NotificationDropped()
    SubscriptionsActive(count int)

    ErrorOccurred(category ErrorCategory, operation string)

    HealthCheckStarted()
    HealthCheckCompleted(success bool)
}
```

### Example: Prometheus Integration

```go
type PrometheusMetrics struct {
    connectionAttempts  prometheus.Counter
    connectionSuccesses prometheus.Counter
    operationDuration   *prometheus.HistogramVec
    // ... other metrics
}

func (m *PrometheusMetrics) ConnectionAttempts() {
    m.connectionAttempts.Inc()
}

func (m *PrometheusMetrics) OperationCompleted(operation string, duration time.Duration, err error) {
    m.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
// ... implement other methods
```

## Complete Example

See [examples/observability/main.go](../examples/observability/main.go) for a complete working example that demonstrates:

- Setting up structured logging
- Enabling metrics collection
- Handling classified errors
- Monitoring connection state changes
- Viewing metrics snapshots

## Best Practices

1. **Use structured logging in production** - JSON format makes logs easy to parse and analyze
2. **Monitor error categories** - Different categories may require different handling strategies
3. **Track operation metrics** - Identify slow operations and failure patterns
4. **Set up health check monitoring** - Detect connection issues early
5. **Use no-op implementations by default** - Minimal overhead when observability is not needed

## Performance Considerations

- Default logger and metrics are no-op implementations with minimal overhead
- Logging at DEBUG level can be verbose; use INFO or WARN in production
- In-memory metrics collector is thread-safe but accumulates data (suitable for short-lived applications)
- For long-running applications, implement custom metrics that export to external systems
