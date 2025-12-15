# goadstc

A production-quality Go library for TwinCAT ADS/AMS communication over TCP.

## What is ADS/AMS?

**ADS (Automation Device Specification)** is a device- and fieldbus-independent protocol developed by Beckhoff for communication with TwinCAT devices. **AMS (Automation Message Specification)** provides the underlying message routing and addressing layer.

This library implements the ADS/AMS protocol specification for TCP transport, enabling Go applications to communicate with TwinCAT PLCs and other ADS-compatible devices.

## Features

### Core Protocol

- ✅ Full AMS/TCP and AMS header encoding/decoding
- ✅ Core ADS commands: Read, Write, ReadWrite, ReadState, ReadDeviceInfo, WriteControl
- ✅ Symbol resolution: Read/write PLC variables by name
- ✅ Symbol table caching: Automatic symbol upload and parsing
- ✅ ADS notifications for real-time variable monitoring
- ✅ Correct binary protocol implementation (little-endian, exact field sizes)
- ✅ Type-safe Go API with functional options
- ✅ Zero external dependencies (standard library only)
- ✅ Production-ready error handling

### Advanced Features

- ✅ **34 Type-Safe Methods**: Read/write all TwinCAT data types with native Go types
- ✅ **Automatic Type Discovery**: Fetch struct definitions directly from PLC
- ✅ **Struct Field Access**: Direct access to struct fields using dot notation
- ✅ **Array Element Access**: Access array elements with bracket notation
- ✅ **Complex Arrays**: Support for struct arrays and nested field access
- ✅ **Time/Date Types**: Native Go `time.Time` and `time.Duration` conversions
- ✅ **Unicode Strings**: Full WSTRING support with UTF-16LE encoding
- ✅ **Symbol-Based Notifications**: Subscribe to PLC variables by name

## What This Library Does NOT Support

- UDP transport (TCP only)
- Router/routing table management (direct TCP connection only)

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
    "github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
    // Create client with target configuration
    plcIP := "10.10.0.3:48898"
    plcNetID := ams.NetID{10, 0, 10, 20, 1, 1}
    pcNetID := ams.NetID{10, 10, 0, 10, 1, 1}

    client, err := goadstc.New(
        goadstc.WithTarget(plcIP),
        goadstc.WithAMSNetID(plcNetID),
        goadstc.WithSourceNetID(pcNetID),
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

    // Type-safe operations (recommended)
    value, err := client.ReadUint16(ctx, "MAIN.counter")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Counter: %d\n", value)

    // Write a value
    err = client.WriteUint16(ctx, "MAIN.counter", 42)
    if err != nil {
        log.Fatal(err)
    }

    // Access struct fields
    temperature, err := client.ReadFloat32(ctx, "MAIN.sensor.temperature")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Temperature: %.2f°C\n", temperature)

    // Access array elements
    arrayValue, err := client.ReadInt32(ctx, "MAIN.dataArray[5]")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Array[5]: %d\n", arrayValue)
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

**Basic Operations:**

- `ReadDeviceInfo(ctx)` - Read device name and version
- `Read(ctx, indexGroup, indexOffset, length)` - Read data from device
- `Write(ctx, indexGroup, indexOffset, data)` - Write data to device
- `ReadState(ctx)` - Read ADS and device state
- `WriteControl(ctx, adsState, deviceState, data)` - Change ADS state (start/stop/reset PLC)
- `ReadWrite(ctx, indexGroup, indexOffset, readLength, writeData)` - Combined read/write operation

**Symbol Resolution:**

- `RefreshSymbols(ctx)` - Download and cache symbol table from PLC
- `GetSymbol(name)` - Get cached symbol information
- `ListSymbols(ctx)` - List all symbols in cache
- `FindSymbols(ctx, pattern)` - Search symbols by pattern
- `ReadSymbol(ctx, name)` - Read PLC variable by name (auto-loads symbols)
- `WriteSymbol(ctx, name, data)` - Write PLC variable by name

**Notifications:**

- `Subscribe(ctx, opts)` - Create a notification subscription for real-time PLC data monitoring
- `SubscribeSymbol(ctx, symbolName, opts)` - Subscribe to a symbol by name

### Type-Safe Operations (Recommended)

All methods automatically resolve symbols and handle type conversions:

**Boolean:**

- `ReadBool(ctx, symbolName) (bool, error)`
- `WriteBool(ctx, symbolName, value bool) error`

**Integers (Signed):**

- `ReadInt8/ReadInt16/ReadInt32/ReadInt64(ctx, symbolName)`
- `WriteInt8/WriteInt16/WriteInt32/WriteInt64(ctx, symbolName, value)`

**Integers (Unsigned):**

- `ReadUint8/ReadUint16/ReadUint32/ReadUint64(ctx, symbolName)`
- `WriteUint8/WriteUint16/WriteUint32/WriteUint64(ctx, symbolName, value)`

**Floating Point:**

- `ReadFloat32/ReadFloat64(ctx, symbolName)`
- `WriteFloat32/WriteFloat64(ctx, symbolName, value)`

**Strings:**

- `ReadString(ctx, symbolName) (string, error)` - ASCII strings
- `WriteString(ctx, symbolName, value string) error`
- `ReadWString(ctx, symbolName) (string, error)` - Unicode (UTF-16LE)
- `WriteWString(ctx, symbolName, value string) error`

**Time/Date Types:**

- `ReadTime(ctx, symbolName) (time.Duration, error)` - TIME (milliseconds)
- `WriteTime(ctx, symbolName, value time.Duration) error`
- `ReadDate(ctx, symbolName) (time.Time, error)` - DATE (Unix timestamp)
- `WriteDate(ctx, symbolName, value time.Time) error`
- `ReadTimeOfDay(ctx, symbolName) (time.Duration, error)` - TIME_OF_DAY (ms since midnight)
- `WriteTimeOfDay(ctx, symbolName, value time.Duration) error`
- `ReadDateAndTime(ctx, symbolName) (time.Time, error)` - DATE_AND_TIME (Unix timestamp)
- `WriteDateAndTime(ctx, symbolName, value time.Time) error`

### Advanced Symbol Access

**Struct Field Access** (dot notation):

```go
// Direct field access
value, err := client.ReadInt32(ctx, "MAIN.myStruct.field1")
err = client.WriteFloat32(ctx, "MAIN.sensor.temperature", 25.5)
```

**Array Element Access** (bracket notation):

```go
// Array indexing
value, err := client.ReadUint16(ctx, "MAIN.dataArray[5]")
err = client.WriteInt32(ctx, "MAIN.buffer[10]", 42)
```

**Combined Access** (arrays of structs):

```go
// Access struct field in array element
value, err := client.ReadFloat32(ctx, "MAIN.sensors[2].temperature")
err = client.WriteUint16(ctx, "MAIN.devices[0].status", 1)
```

**Automatic Struct Parsing:**

```go
// Automatically fetch type info from PLC and parse all fields
structData, err := client.ReadStructAsMap(ctx, "MAIN.myStruct")
if err != nil {
    log.Fatal(err)
}

// Access parsed fields
field1 := structData["field1"].(int32)
field2 := structData["temperature"].(float32)
nested := structData["subStruct"].(map[string]interface{})
```

## Usage Examples

### Type-Safe Operations

```go
// Read various data types
boolVal, err := client.ReadBool(ctx, "MAIN.running")
intVal, err := client.ReadInt32(ctx, "MAIN.counter")
floatVal, err := client.ReadFloat32(ctx, "MAIN.temperature")
strVal, err := client.ReadString(ctx, "MAIN.message")

// Write various data types
err = client.WriteBool(ctx, "MAIN.enable", true)
err = client.WriteInt32(ctx, "MAIN.setpoint", 100)
err = client.WriteFloat32(ctx, "MAIN.pressure", 1.5)
err = client.WriteString(ctx, "MAIN.status", "OK")

// Time and date operations
duration := 5*time.Second + 250*time.Millisecond
err = client.WriteTime(ctx, "MAIN.delay", duration)

timestamp := time.Now()
err = client.WriteDateAndTime(ctx, "MAIN.lastUpdate", timestamp)

// Unicode strings
err = client.WriteWString(ctx, "MAIN.unicodeText", "Hello 世界 🌍!")
text, err := client.ReadWString(ctx, "MAIN.unicodeText")
```

### Struct and Array Access

```go
// Struct field access (dot notation)
temp, err := client.ReadFloat32(ctx, "MAIN.sensor.temperature")
err = client.WriteUint16(ctx, "MAIN.config.timeout", 5000)

// Array element access (bracket notation)
value, err := client.ReadInt32(ctx, "MAIN.dataBuffer[10]")
err = client.WriteUint16(ctx, "MAIN.statusArray[5]", 42)

// Struct arrays (combined notation)
sensorTemp, err := client.ReadFloat32(ctx, "MAIN.sensors[2].temperature")
err = client.WriteString(ctx, "MAIN.devices[0].name", "Device1")

// Automatic struct parsing (fetches type info from PLC)
data, err := client.ReadStructAsMap(ctx, "MAIN.myStruct")
if err != nil {
    log.Fatal(err)
}

// Access parsed fields by name
field1 := data["field1"].(int32)
field2 := data["temperature"].(float32)
field3 := data["name"].(string)

// Works with struct arrays too!
structData, err := client.ReadStructAsMap(ctx, "MAIN.structArray[3]")
```

### Notifications (Real-Time Monitoring)

```go
// Subscribe to a variable by name (easiest)
sub, err := client.SubscribeSymbol(ctx, "MAIN.counter", goadstc.SymbolNotificationOptions{
    TransmissionMode: ads.TransModeOnChange,  // Notify on change
    MaxDelay:         100 * time.Millisecond,
    CycleTime:        50 * time.Millisecond,
})
if err != nil {
    log.Fatal(err)
}
defer sub.Close()

// Process notifications
go func() {
    for notif := range sub.Notifications() {
        value := binary.LittleEndian.Uint32(notif.Data)
        fmt.Printf("Counter changed to %d at %s\n", value, notif.Timestamp)
    }
}()

// Supported transmission modes:
// - TransModeCyclic: Send at fixed intervals
// - TransModeOnChange: Send only when value changes
// - TransModeCyclicOnChange: Combination of both
```

### Symbol Table Operations

```go
// List all symbols
symbols, err := client.ListSymbols(ctx)
for _, sym := range symbols {
    fmt.Printf("%s: %s (%d bytes)\n", sym.Name, sym.Type.Name, sym.Size)
}

// Search for symbols
mainSymbols, err := client.FindSymbols(ctx, "MAIN")
for _, sym := range mainSymbols {
    fmt.Printf("Found: %s\n", sym.Name)
}

// Get specific symbol info
symbol, err := client.GetSymbol("MAIN.counter")
fmt.Printf("Type: %s, Size: %d, IndexGroup: 0x%X\n",
    symbol.Type.Name, symbol.Size, symbol.IndexGroup)

// Refresh symbol table (if PLC program changes)
err = client.RefreshSymbols(ctx)
```

### Low-Level Operations

```go
// Direct memory access (if you need it)
data, err := client.Read(ctx, 0x4020, 0, 4) // Read 4 bytes from %M0
err = client.Write(ctx, 0x4020, 0, []byte{0x01, 0x02, 0x03, 0x04})

// Combined read-write
result, err := client.ReadWrite(ctx, 0xF003, 0, 4, []byte("MAIN.var\x00"))

// PLC control
state, err := client.ReadState(ctx)
err = client.WriteControl(ctx, ads.StateRun, 0, nil) // Start PLC
```

## Data Type Mapping

| TwinCAT Type         | Go Type         | Size     | Read Method       | Write Method       |
| -------------------- | --------------- | -------- | ----------------- | ------------------ |
| BOOL                 | `bool`          | 1 byte   | `ReadBool`        | `WriteBool`        |
| BYTE, USINT, UINT8   | `uint8`         | 1 byte   | `ReadUint8`       | `WriteUint8`       |
| SINT, INT8           | `int8`          | 1 byte   | `ReadInt8`        | `WriteInt8`        |
| WORD, UINT, UINT16   | `uint16`        | 2 bytes  | `ReadUint16`      | `WriteUint16`      |
| INT, INT16           | `int16`         | 2 bytes  | `ReadInt16`       | `WriteInt16`       |
| DWORD, UDINT, UINT32 | `uint32`        | 4 bytes  | `ReadUint32`      | `WriteUint32`      |
| DINT, INT32          | `int32`         | 4 bytes  | `ReadInt32`       | `WriteInt32`       |
| LWORD, ULINT, UINT64 | `uint64`        | 8 bytes  | `ReadUint64`      | `WriteUint64`      |
| LINT, INT64          | `int64`         | 8 bytes  | `ReadInt64`       | `WriteInt64`       |
| REAL, FLOAT          | `float32`       | 4 bytes  | `ReadFloat32`     | `WriteFloat32`     |
| LREAL, DOUBLE        | `float64`       | 8 bytes  | `ReadFloat64`     | `WriteFloat64`     |
| STRING               | `string`        | Variable | `ReadString`      | `WriteString`      |
| WSTRING              | `string`        | Variable | `ReadWString`     | `WriteWString`     |
| TIME                 | `time.Duration` | 4 bytes  | `ReadTime`        | `WriteTime`        |
| DATE                 | `time.Time`     | 4 bytes  | `ReadDate`        | `WriteDate`        |
| TIME_OF_DAY, TOD     | `time.Duration` | 4 bytes  | `ReadTimeOfDay`   | `WriteTimeOfDay`   |
| DATE_AND_TIME, DT    | `time.Time`     | 4 bytes  | `ReadDateAndTime` | `WriteDateAndTime` |

### Common Index Groups (Low-Level Access)

| Index Group  | Description               |
| ------------ | ------------------------- |
| `0x00004020` | PLC memory (%M)           |
| `0x0000F020` | Physical inputs (%I)      |
| `0x0000F030` | Physical outputs (%Q)     |
| `0xF003`     | Get symbol handle by name |
| `0xF006`     | Release symbol handle     |
| `0xF00B`     | Upload symbol table       |
| `0xF00C`     | Get symbol upload info    |
| `0xF010`     | Get data type upload info |
| `0xF011`     | Upload data type info     |

## Examples

See the [`examples/`](examples/) directory for complete working examples:

- [`examples/arrays/`](examples/arrays/) - Array element access and struct arrays
- [`examples/comprehensive/`](examples/comprehensive/) - Complete feature demonstration
- [`examples/control/`](examples/control/) - PLC control operations
- [`examples/notifications/`](examples/notifications/) - Real-time notifications
- [`examples/structs/`](examples/structs/) - Automatic struct parsing
- [`examples/symbol-notifications/`](examples/symbol-notifications/) - Symbol-based notifications
- [`examples/symbols/`](examples/symbols/) - Symbol table operations
- [`examples/timedate/`](examples/timedate/) - Time and date type operations
- [`examples/typesafe/`](examples/typesafe/) - Type-safe read/write operations

## Project Structure

```
/
├── client.go                  # Core client API
├── client_types.go            # Type-safe read/write methods
├── client_structs.go          # Struct parsing and type discovery
├── client_notifications.go    # Notification subscriptions
├── subscription.go            # Subscription management
├── internal/
│   ├── ams/                   # AMS protocol implementation
│   ├── ads/                   # ADS command handling
│   ├── symbols/               # Symbol table parsing
│   └── transport/             # TCP transport layer
├── examples/                  # Usage examples
└── tests/                     # Test suite
```

## Development

### Building

```bash
go build ./...
```

### Running Examples

```bash
# Configure your PLC connection in the example
go run ./examples/typesafe/.
go run ./examples/arrays/.
go run ./examples/timedate/.
```

### Testing

```bash
go test ./...
```

## Protocol Implementation

This library implements the TwinCAT ADS/AMS protocol according to the official Beckhoff specification:

- **AMS/TCP Header**: 6 bytes (reserved + length)
- **AMS Header**: 32 bytes (routing and control information)
- **ADS Data**: Variable length payload
- **Byte Order**: Little-endian for all multi-byte fields
- **Transport**: TCP only (UDP not supported)
- **Automatic Type Discovery**: Fetches struct definitions from PLC using command 0xF011
- **Symbol Resolution**: Caches symbol table for fast lookups

## Limitations

- **UDP Transport**: Not supported (TCP only)
- **Router Management**: Direct TCP connection only, no routing table management
- **Multi-Dimensional Arrays**: Currently supports single dimension (e.g., `array[5]`)

## Contributing

Contributions are welcome! Please ensure:

- Code follows Go conventions
- All examples compile and run
- Documentation is updated for new features

## License

MIT

## References

- [TwinCAT ADS/AMS Specification](https://infosys.beckhoff.com/english.php?content=../content/1033/tc3_ads_intro/index.html)
- [Beckhoff Information System](https://infosys.beckhoff.com/)

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/mrpasztoradam/goadstc).
