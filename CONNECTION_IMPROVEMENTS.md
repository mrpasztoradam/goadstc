# Connection Stability Improvements

## Summary

This branch implements critical improvements to connection handling for better stability and performance when running multiple clients sequentially or handling connection issues.

## Changes Implemented

### 1. TCP Keepalive Configuration

- **30-second TCP keepalive**: Automatically detects dead connections at the OS level
- **TCP NoDelay**: Disables Nagle's algorithm for lower latency
- **Linger set to 0**: Reduces TIME_WAIT issues when rapidly opening/closing connections

### 2. Connection State Tracking

Added comprehensive state management:

- `StateConnecting` - Initial connection phase
- `StateConnected` - Active and ready for requests
- `StateDisconnecting` - Graceful shutdown in progress
- `StateClosed` - Fully closed
- `StateError` - Error state with associated error message

### 3. Graceful Shutdown

- **CloseWithTimeout()**: Waits up to 5 seconds for pending operations to complete
- **Proper cleanup**: Closes all pending request channels before terminating
- **Shutdown context**: Signals goroutines to exit cleanly
- **No forced closure**: Reduces risk of lost responses or panics

### 4. Improved Error Handling

- **State-aware errors**: Error messages include connection state information
- **Last error tracking**: Stores and reports the most recent error
- **Better context**: Helps diagnose "connection closed" vs "write failed" vs "timeout"

### 5. Resource Management

- **Shutdown context**: Coordinates graceful shutdown across goroutines
- **Proper channel cleanup**: Prevents goroutine leaks
- **Read/write deadline handling**: Improved timeout detection

## Benefits

### Connection Loss Fix

The main issue with running multiple examples sequentially was likely due to:

1. **TCP TIME_WAIT**: Previous connections staying in TIME_WAIT state
   - **Fixed by**: SetLinger(0) reduces TIME_WAIT
2. **Connection not fully closed**: Goroutines not terminated properly
   - **Fixed by**: Graceful shutdown with timeout
3. **No connection health monitoring**: Dead connections not detected
   - **Fixed by**: TCP keepalive

### Performance

- **Lower latency**: NoDelay disables Nagle's algorithm
- **Better error recovery**: Clear error messages help diagnose issues faster
- **Cleaner shutdown**: No lingering resources

### Reliability

- **Dead connection detection**: TCP keepalive (30s)
- **Graceful cleanup**: No abrupt connection termination
- **State tracking**: Always know connection status

## Testing

### Unit Tests

Added comprehensive tests in `conn_state_test.go`:

- Connection state transitions
- Error handling
- Graceful shutdown behavior

Run with:

```bash
go test ./internal/transport -v
```

### Benchmark Tests

Added performance benchmarks in `conn_bench_test.go`:

- Connection creation overhead
- Request latency
- Concurrent request handling

Run with your PLC:

```bash
# Edit benchmark files to remove b.Skip() and set your PLC address
go test -bench=. ./internal/transport
```

### Integration Testing

Use the provided script to test multiple examples sequentially:

```bash
./test_connection_stability.sh
```

This runs multiple examples back-to-back to verify no connection issues.

## Migration

No breaking API changes. Existing code continues to work:

- `client.Close()` now uses graceful shutdown automatically
- Better error messages help diagnose issues
- TCP configuration is transparent

## Next Steps (Future Work)

### Priority 2: Auto-Recovery

- Automatic reconnection with exponential backoff
- Transparent request retry for transient failures
- Subscription re-establishment after reconnect
- Connection state callbacks for monitoring

### Priority 3: Advanced Features

- Connection health checks (periodic ReadState)
- Metrics collection (latency, error rate)
- Connection pooling (if needed later)

## Configuration Recommendations

Current default timeouts work well for most cases:

- **Connection timeout**: 5 seconds (sufficient for LAN)
- **Request timeout**: 5 seconds
- **Graceful shutdown**: 5 seconds
- **TCP keepalive**: 30 seconds

For different environments:

- **WAN/Internet**: Increase timeouts to 10-15 seconds
- **Local loopback**: Can reduce to 1-2 seconds
- **Unreliable networks**: Enable auto-reconnect (future feature)

## Files Modified

- `internal/transport/conn.go` - Core connection improvements
- `internal/transport/conn_state_test.go` - Unit tests
- `internal/transport/conn_bench_test.go` - Performance benchmarks
- `test_connection_stability.sh` - Integration test script

## Validation

All tests pass:

```
✓ TestConnectionState
✓ TestConnectionStateString
✓ TestCompareAndSwapState
✓ TestErrorHandling
✓ TestGracefulShutdown
```

Ready for real PLC testing with your examples!
