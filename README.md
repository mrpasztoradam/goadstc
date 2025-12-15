# goadstc

A production-quality Go library for TwinCAT ADS/AMS communication over TCP.

## What is ADS/AMS?

**ADS (Automation Device Specification)** is a device- and fieldbus-independent protocol developed by Beckhoff for communication with TwinCAT devices. **AMS (Automation Message Specification)** provides the underlying message routing and addressing layer.

This library implements the ADS/AMS protocol specification for TCP transport, enabling Go applications to communicate with TwinCAT PLCs and other ADS-compatible devices.

## Features

- Full AMS/TCP and AMS header encoding/decoding
- Core ADS commands: Read, Write, ReadWrite, ReadState, ReadDeviceInfo
- Correct binary protocol implementation (little-endian, exact field sizes)
- Type-safe Go API with functional options
- Zero external dependencies (standard library only)
- Production-ready error handling

## What This Library Does NOT Support

- UDP transport (TCP only)
- High-level PLC abstractions beyond ADS protocol
- Router/routing table management
- TwinCAT-specific symbol resolution

## Installation

```bash
go get github.com/mrpasztoradam/goadstc
```

## Project Structure

```
/
├── client.go              # Public API
├── internal/
│   ├── ams/              # AMS protocol implementation
│   ├── ads/              # ADS command handling
│   └── transport/        # TCP transport layer
├── tests/                # Test suite (gitignored)
├── examples/             # Usage examples
└── testdata/             # Test fixtures
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/mrpasztoradam/goadstc"
)

func main() {
    // Create client with target configuration
    client, err := goadstc.New(
        goadstc.WithTarget("192.168.1.100:48898"),
        goadstc.WithAMSNetID([6]byte{192, 168, 1, 100, 1, 1}),
        goadstc.WithAMSPort(851),
        goadstc.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Read device info
    info, err := client.ReadDeviceInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Device: %s (v%d.%d.%d)\n",
        info.Name, info.MajorVersion, info.MinorVersion, info.VersionBuild)

    // Read data from PLC
    data, err := client.Read(ctx, 0x4020, 0, 4) // Read 4 bytes from %M0
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Read data: %v\n", data)
}
```

## API Overview

### Client Configuration Options

- `WithTarget(address)` - Target TCP address (required)
- `WithAMSNetID(netID)` - Target AMS NetID (required)
- `WithAMSPort(port)` - Target AMS port (default: 851)
- `WithSourceNetID(netID)` - Source AMS NetID (optional)
- `WithSourcePort(port)` - Source AMS port (default: 32905)
- `WithTimeout(duration)` - Request timeout (default: 5s)

### Core Methods

- `ReadDeviceInfo(ctx)` - Read device name and version
- `Read(ctx, indexGroup, indexOffset, length)` - Read data from device
- `Write(ctx, indexGroup, indexOffset, data)` - Write data to device
- `ReadState(ctx)` - Read ADS and device state
- `WriteControl(ctx, adsState, deviceState, data)` - Change ADS state (start/stop/reset PLC)
- `ReadWrite(ctx, indexGroup, indexOffset, readLength, writeData)` - Combined read/write operation
- `Subscribe(ctx, opts)` - Create a notification subscription for real-time PLC data monitoring

### Notifications

The library supports ADS notifications for monitoring PLC variables in real-time:

```go
// Subscribe to variable changes
sub, err := client.Subscribe(ctx, goadstc.NotificationOptions{
    IndexGroup:       ads.IndexGroupPLCMemory,
    IndexOffset:      0x1000,
    Length:           4,
    TransmissionMode: ads.TransModeOnChange,  // Notify on change
    MaxDelay:         100 * time.Millisecond,
    CycleTime:        50 * time.Millisecond,
})
if err != nil {
    log.Fatal(err)
}
defer sub.Close()

// Process notifications
for notif := range sub.Notifications() {
    fmt.Printf("Value at %s: %v\n", notif.Timestamp, notif.Data)
}
```

Supported transmission modes:

- `TransModeCyclic` - Send notifications at fixed intervals
- `TransModeOnChange` - Send only when value changes
- `TransModeCyclicOnChange` - Combination of both

See [examples/notifications](examples/notifications/main.go) for a complete example.

### Common Index Groups

- `0x00004020` - PLC memory (%M)
- `0x0000F020` - Physical inputs (%I)
- `0x0000F030` - Physical outputs (%Q)

## Development

### Running Tests

Tests are located in the `tests/` directory (gitignored by default):

```bash
go test ./tests/...
```

### Building

```bash
go build ./...
```

## Protocol Implementation

This library implements the TwinCAT ADS/AMS protocol according to the official Beckhoff specification:

- **AMS/TCP Header**: 6 bytes (reserved + length)
- **AMS Header**: 32 bytes (routing and control information)
- **ADS Data**: Variable length payload
- **Byte Order**: Little-endian for all multi-byte fields
- **Transport**: TCP only (UDP not supported)

## License

MIT

## References

- [TwinCAT ADS/AMS Specification](https://infosys.beckhoff.com/english.php?content=../content/1033/tc3_ads_intro/index.html)
