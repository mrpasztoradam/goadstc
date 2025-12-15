# AI Agent Development Guide for goadstc

This guide provides essential context and patterns for AI agents working on this ADS/AMS client library.

## Project Overview

**goadstc** is a production-quality Go library for TwinCAT ADS/AMS communication over TCP. It enables Go applications to communicate with Beckhoff PLCs.

### Core Principles

- **Zero external dependencies** - Standard library only
- **Type safety** - Strong typing throughout, no `interface{}` abuse
- **Production ready** - Comprehensive error handling, connection stability, observability
- **Opt-in complexity** - Advanced features (logging, metrics, reconnection) are optional with zero overhead by default

### Architecture

```
/
├── client*.go           # Public API (main client, notifications, types, structs)
├── subscription.go      # Notification subscription management
├── logging.go          # Structured logging interface
├── metrics.go          # Metrics collection interface
├── errors.go           # Error classification system
├── version.go          # Library versioning
├── internal/           # Protocol implementation (not exported)
│   ├── ads/           # ADS command structures and constants
│   ├── ams/           # AMS packet encoding/decoding
│   ├── symbols/       # Symbol table parsing
│   └── transport/     # TCP connection management
├── examples/          # Usage examples (one per feature)
└── tests/            # Integration tests
```

## Domain Knowledge: ADS/AMS Protocol

### Key Concepts

- **AMS (Automation Message Specification)**: Routing layer with NetID addressing (6-byte identifier like 192.168.1.1.1.1)
- **ADS (Automation Device Specification)**: Command layer for read/write operations
- **Index Group/Offset**: Memory addressing scheme (not byte addresses - semantic addressing)
- **Symbols**: Named PLC variables that must be resolved to IndexGroup/Offset
- **Notifications**: Push-based real-time updates from PLC to client

### Protocol Quirks

- **Little-endian** encoding throughout
- **TCP keepalive required** - PLCs may silently drop idle connections
- **Symbol table upload** - Large XML-like data, requires extended timeouts
- **Config mode** - PLC may be unresponsive during configuration (port not found errors)
- **Notification handles** - Server-assigned, must be tracked per subscription

## Coding Conventions

### File Organization

- **client.go**: Core operations (Read, Write, ReadState, connection lifecycle)
- **client\_[feature].go**: Feature-specific methods (notifications, types, structs)
- **internal/** packages: Protocol implementation details
- One example per feature in `examples/[feature]/main.go`

### Naming Patterns

- **Public types**: `Client`, `Subscription`, `DeviceInfo`, `NotificationOptions`
- **Internal types**: Lower visibility, package-scoped when possible
- **Options pattern**: Functional options for configuration (`WithTarget`, `WithLogger`, etc.)
- **Context first**: All I/O operations take `context.Context` as first parameter

### Error Handling

```go
// Always classify errors for observability
if err != nil {
    c.logger.Error("operation failed", "error", err)
    c.metrics.OperationCompleted("operation", time.Since(start), err)
    ce := ClassifyError(err, "operation")
    c.metrics.ErrorOccurred(ce.Category, "operation")
    return ce
}
```

### Observability Pattern

All I/O operations must:

1. Start with metrics: `c.metrics.OperationStarted("operation")`
2. Log at appropriate level (Debug for details, Error for failures)
3. Track timing: `start := time.Now()`
4. Complete with metrics: `c.metrics.OperationCompleted("operation", duration, err)`
5. Classify errors: `ce := ClassifyError(err, "operation")`

### Thread Safety

- Use `sync.RWMutex` for read-heavy maps (subscriptions, symbol table)
- Use `atomic` for counters and flags
- Document locking order to prevent deadlocks
- Keep critical sections small

## Testing Requirements

### Test Coverage

- All exported functions must have tests
- Use table-driven tests for multiple scenarios
- Include edge cases (nil inputs, empty strings, boundary values)

### Example Pattern

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   Type
        want    Type
        wantErr bool
    }{
        {"valid case", validInput, expectedOutput, false},
        {"error case", invalidInput, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Git Workflow

### Commit Message Format

```
<type>: <short summary>

<detailed description>

<bulleted list of changes>
```

**Types**: `feat`, `fix`, `refactor`, `test`, `docs`, `perf`

### Good Commit Examples

```
feat: add structured logging and metrics

Add comprehensive observability features:
- Structured logging using log/slog
- Error classification system
- Metrics collection interface
```

### Branch Strategy

- Feature branches: `feat/feature-name`
- Bug fixes: `fix/issue-description`
- Main branch: Keep stable and tested

## Common Patterns to Follow

### 1. Adding New Operations

```go
func (c *Client) NewOperation(ctx context.Context, params) (result, error) {
    start := time.Now()
    c.metrics.OperationStarted("new_operation")
    c.logger.Debug("performing operation", "param", value)

    // Build request
    req := ads.NewRequest{...}
    reqData, _ := req.MarshalBinary()

    // Send request
    respPacket, err := c.sendRequest(ctx, ads.CmdSomething, reqData)
    if err != nil {
        c.logger.Error("operation failed", "error", err)
        c.metrics.OperationCompleted("new_operation", time.Since(start), err)
        ce := ClassifyError(err, "new_operation")
        c.metrics.ErrorOccurred(ce.Category, "new_operation")
        return nil, ce
    }

    // Parse response
    var resp ads.NewResponse
    if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
        c.logger.Error("unmarshal failed", "error", err)
        c.metrics.OperationCompleted("new_operation", time.Since(start), err)
        c.metrics.ErrorOccurred(ErrorCategoryProtocol, "new_operation")
        return nil, err
    }

    // Check ADS error
    if resp.Result != 0 {
        adsErr := ads.Error(resp.Result)
        c.logger.Error("ADS error", "error", adsErr)
        c.metrics.OperationCompleted("new_operation", time.Since(start), adsErr)
        c.metrics.ErrorOccurred(ErrorCategoryADS, "new_operation")
        return nil, NewADSError("new_operation", adsErr)
    }

    c.metrics.OperationCompleted("new_operation", time.Since(start), nil)
    c.logger.Debug("operation completed", "duration", time.Since(start))
    return result, nil
}
```

### 2. Adding New Client Options

```go
// WithFeature description of what it does (optional).
func WithFeature(value Type) Option {
    return func(c *clientConfig) error {
        if /* validation */ {
            return fmt.Errorf("goadstc: validation message")
        }
        c.feature = value
        return nil
    }
}
```

### 3. Adding Examples

Each example should:

- Be runnable with `go run examples/feature/main.go`
- Use environment variables for configuration
- Demonstrate one specific feature clearly
- Include error handling
- Show expected output in comments

## Things to NEVER Change

### Breaking Changes to Avoid

1. **Public API signatures** - Maintain backward compatibility
2. **Default behavior** - Opt-in for new features, no breaking existing code
3. **Error types** - Don't change existing error classifications
4. **Protocol implementation** - ADS/AMS encoding must match specification exactly
5. **Zero dependencies** - Never add external dependencies

### Performance Considerations

- No-op implementations must have zero overhead (check assembly if needed)
- Avoid allocations in hot paths
- Use `sync.Pool` for frequently allocated objects if needed
- Profile before optimizing

### Security

- Validate all inputs from untrusted sources
- Don't log sensitive data (use `"password", "***"` pattern)
- Use constant-time comparison for security-sensitive operations

## File Modification Checklist

When modifying code, ensure:

- [ ] Tests added/updated and passing (`go test ./...`)
- [ ] Examples updated if public API changed
- [ ] Documentation updated (inline comments, README if needed)
- [ ] CHANGELOG.md updated for user-facing changes
- [ ] Error handling includes logging and metrics
- [ ] Code formatted (`go fmt ./...`)
- [ ] No new external dependencies
- [ ] Backward compatible (or documented breaking change)

## Quick Reference

### Build & Test

```bash
go build ./...          # Build all packages
go test ./...          # Run all tests
go test -v -run Test   # Run specific test
go fmt ./...           # Format code
```

### Common Mistakes to Avoid

- ❌ Forgetting context in I/O operations
- ❌ Not logging errors before returning
- ❌ Missing metrics tracking
- ❌ Blocking operations without timeout
- ❌ Mutations without proper locking
- ❌ Using `interface{}` instead of proper types
- ❌ Not classifying errors

### Getting Help

- Read the [TwinCAT ADS specification](https://infosys.beckhoff.com/english.php?content=../content/1033/tc3_ads_intro/index.html)
- Check existing examples in `examples/`
- Review similar operations in `client*.go`
- Test against real PLC or use mocks in `internal/transport/`

## Future Development Areas

### Planned Features

- Prometheus metrics exporter
- Distributed tracing (OpenTelemetry)
- Connection pooling
- Batch operations
- More symbol types

### Out of Scope

- UDP transport (TCP only by design)
- Router management (direct connection only)
- GUI/CLI tools (library only)
- Non-Beckhoff protocols

---

**Remember**: This is a production library. Reliability, observability, and backward compatibility are paramount.
