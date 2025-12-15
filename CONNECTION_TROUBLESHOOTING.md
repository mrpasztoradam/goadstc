# Connection Troubleshooting

## Issue: "connection reset by peer"

This error means the PLC is rejecting the connection, usually because the AMS NetID configuration doesn't match.

## Quick Fix Steps

### 1. Find Your PLC's Actual AMS NetID

In TwinCAT System Manager on the PLC:
- Open **System** → **Local Router** 
- Look for **AMS Net Id** (e.g., `10.10.0.3.1.1` or similar)

### 2. Update Examples with Correct NetID

The current examples use:
```go
plcNetID := ams.NetID{10, 0, 10, 20, 1, 1}  // ← This might be wrong!
```

Replace with your actual PLC NetID. For example, if your PLC shows `10.10.0.3.1.1`:
```go
plcNetID := ams.NetID{10, 10, 0, 3, 1, 1}
```

### 3. Common NetID Patterns

Most TwinCAT installations use one of these patterns:

**Pattern 1: Based on IP Address**
- PLC IP: `10.10.0.3`
- PLC NetID: `10.10.0.3.1.1` → `{10, 10, 0, 3, 1, 1}`

**Pattern 2: Custom**
- PLC NetID: Check in TwinCAT (could be anything)

### 4. PC NetID

Your PC (10.10.0.10) NetID should typically match the IP:
```go
pcNetID := ams.NetID{10, 10, 0, 10, 1, 1}
```

Or use zero NetID to let the PLC assign one:
```go
pcNetID := ams.NetID{0, 0, 0, 0, 0, 0}
```

### 5. Testing Different Configurations

Try these configurations in order:

**Config A: IP-based NetID without source**
```go
client, err := goadstc.New(
    goadstc.WithTarget("10.10.0.3:48898"),
    goadstc.WithAMSNetID(ams.NetID{10, 10, 0, 3, 1, 1}),
    goadstc.WithAMSPort(851),
    goadstc.WithTimeout(5*time.Second),
)
```

**Config B: IP-based NetID with source**
```go
client, err := goadstc.New(
    goadstc.WithTarget("10.10.0.3:48898"),
    goadstc.WithAMSNetID(ams.NetID{10, 10, 0, 3, 1, 1}),
    goadstc.WithSourceNetID(ams.NetID{10, 10, 0, 10, 1, 1}),
    goadstc.WithAMSPort(851),
    goadstc.WithTimeout(5*time.Second),
)
```

**Config C: Zero source NetID**
```go
client, err := goadstc.New(
    goadstc.WithTarget("10.10.0.3:48898"),
    goadstc.WithAMSNetID(ams.NetID{10, 10, 0, 3, 1, 1}),
    goadstc.WithSourceNetID(ams.NetID{0, 0, 0, 0, 0, 0}),
    goadstc.WithAMSPort(851),
    goadstc.WithTimeout(5*time.Second),
)
```

### 6. Check AMS Routes

On the PLC, verify your PC is allowed to connect:
- **TwinCAT System Manager** → **SYSTEM** → **Routes**
- Your PC (10.10.0.10) should be listed
- If not, add it: Right-click → **Add Route**

### 7. Quick Test

Create a simple test file `test_connection.go`:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/mrpasztoradam/goadstc"
    "github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
    // Try with IP-based NetID
    client, err := goadstc.New(
        goadstc.WithTarget("10.10.0.3:48898"),
        goadstc.WithAMSNetID(ams.NetID{10, 10, 0, 3, 1, 1}),
        goadstc.WithAMSPort(851),
        goadstc.WithTimeout(5*time.Second),
    )
    if err != nil {
        fmt.Printf("Connection failed: %v\n", err)
        return
    }
    defer client.Close()
    
    ctx := context.Background()
    info, err := client.ReadDeviceInfo(ctx)
    if err != nil {
        fmt.Printf("ReadDeviceInfo failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Success! Device: %s Version: %d.%d.%d\n",
        info.Name, info.MajorVersion, info.MinorVersion, info.VersionBuild)
}
```

Run with: `go run test_connection.go`

## Success Indicators

When it works, you'll see:
```
✅ Success! Device: TwinCAT Version: 3.1.4024
```

Instead of:
```
❌ connection reset by peer
```

## Need Help?

If none of these work:
1. Verify PLC is running: `ping 10.10.0.3`
2. Verify ADS port is open: `telnet 10.10.0.3 48898`
3. Check TwinCAT is in RUN mode
4. Check Windows Firewall on PLC allows ADS traffic
