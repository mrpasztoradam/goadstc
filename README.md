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
- ADS notifications (Add/Delete/Device Notification commands)
- High-level PLC abstractions beyond ADS protocol
- Router/routing table management
- TwinCAT-specific symbol resolution

## Installation

```bash
go get github.com/mrpasztoradam/goadstc
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

## License

MIT

## References

- [TwinCAT ADS/AMS Specification](https://infosys.beckhoff.com/english.php?content=../content/1033/tc3_ads_intro/index.html)
