# GoADS HTTP/WebSocket Middleware Server

REST API and WebSocket server for TwinCAT ADS operations with automatic JSON marshaling.

## Features

- ✅ RESTful API for all symbol operations
- ✅ Batch read/write operations
- ✅ Struct field manipulation
- ✅ Symbol table retrieval
- ✅ Runtime version information
- ✅ PLC state monitoring and control (start/stop/reset)
- ✅ Automatic type detection and conversion
- ✅ CORS support for web clients
- ✅ Swagger documentation at `/swagger-ui/index.html`
- ✅ WebSocket subscriptions for real-time updates

## Quick Start

### 1. Generate Configuration

```bash
go run main.go -generate-config
```

This creates `config.example.yaml`. Copy and edit:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your PLC settings
```

### 2. Start Server

```bash
go run main.go -config config.yaml
```

Or use default config.yaml:

```bash
go run main.go
```

### 3. Test API

Run the comprehensive test script:

```bash
./test-api.sh
```

Or test individual endpoints:

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Server info
curl http://localhost:8080/api/v1/info

# Get symbol table
curl http://localhost:8080/api/v1/symbols

# Read a symbol
curl http://localhost:8080/api/v1/symbols/MAIN.temperature/value

# Write a symbol
curl -X POST http://localhost:8080/api/v1/symbols/MAIN.temperature/value \
  -H "Content-Type: application/json" \
  -d '{"value": 25.5}'

# Batch read
curl -X POST http://localhost:8080/api/v1/symbols/read \
  -H "Content-Type: application/json" \
  -d '{"symbols": ["MAIN.temperature", "MAIN.counter", "MAIN.enabled"]}'

# Batch write
curl -X POST http://localhost:8080/api/v1/symbols/write \
  -H "Content-Type: application/json" \
  -d '{"writes": {"MAIN.temperature": 30.0, "MAIN.enabled": true}}'

# Write struct fields
curl -X POST http://localhost:8080/api/v1/structs/MAIN.myStruct/fields \
  -H "Content-Type: application/json" \
  -d '{"fields": {"temperature": 25.5, "enabled": true, "counter": 42}}'
```

## API Endpoints

### Symbol Operations

| Method | Endpoint                       | Description             |
| ------ | ------------------------------ | ----------------------- |
| GET    | `/api/v1/symbols`              | Get entire symbol table |
| GET    | `/api/v1/symbols/{name}`       | Get symbol metadata     |
| GET    | `/api/v1/symbols/{name}/value` | Read symbol value       |
| POST   | `/api/v1/symbols/{name}/value` | Write symbol value      |
| POST   | `/api/v1/symbols/read`         | Batch read symbols      |
| POST   | `/api/v1/symbols/write`        | Batch write symbols     |

### Struct Operations

| Method | Endpoint                        | Description         |
| ------ | ------------------------------- | ------------------- |
| GET    | `/api/v1/structs/{name}`        | Read entire struct  |
| POST   | `/api/v1/structs/{name}/fields` | Write struct fields |

### System Operations

| Method | Endpoint          | Description                    |
| ------ | ----------------- | ------------------------------ |
| GET    | `/api/v1/health`  | Health check                   |
| GET    | `/api/v1/info`    | Server and PLC info            |
| GET    | `/api/v1/version` | Get runtime version            |
| GET    | `/api/v1/state`   | Get PLC state                  |
| POST   | `/api/v1/control` | Control PLC (start/stop/reset) |

## Configuration

See `config.yaml` for all available options:

- **Server**: Host, port, CORS settings
- **PLC**: Connection parameters
- **Middleware**: Batch size limits, buffer sizes
- **Logging**: Level and format

## Examples

### JavaScript (Browser)

```javascript
// Read symbol
fetch("http://localhost:8080/api/v1/symbols/MAIN.temperature/value")
  .then((res) => res.json())
  .then((data) => console.log("Temperature:", data.value));

// Write symbol
fetch("http://localhost:8080/api/v1/symbols/MAIN.enabled/value", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ value: true }),
});

// Batch operations
fetch("http://localhost:8080/api/v1/symbols/read", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ symbols: ["MAIN.temp", "MAIN.pressure"] }),
})
  .then((res) => res.json())
  .then(console.log);
```

### Python

```python
import requests

# Read symbol
r = requests.get('http://localhost:8080/api/v1/symbols/MAIN.temperature/value')
print(f"Temperature: {r.json()['value']}")

# Write symbol
requests.post(
    'http://localhost:8080/api/v1/symbols/MAIN.enabled/value',
    json={'value': True}
)

# Batch read
r = requests.post(
    'http://localhost:8080/api/v1/symbols/read',
    json={'symbols': ['MAIN.temp', 'MAIN.pressure']}
)
print(r.json())
```

## Response Formats

### Success Response

```json
{
  "success": true,
  "symbol": "MAIN.temperature",
  "value": 25.5,
  "type": "float32"
}
```

### Error Response

```json
{
  "error": {
    "code": "SYMBOL_NOT_FOUND",
    "message": "Symbol 'MAIN.invalid' not found in PLC",
    "details": {
      "symbol": "MAIN.invalid"
    }
  }
}
```

### Batch Response

```json
{
  "success": true,
  "data": {
    "MAIN.temperature": 25.5,
    "MAIN.counter": 42,
    "MAIN.enabled": true
  },
  "errors": {}
}
```

## Command Line Options

```bash
go run main.go [options]

Options:
  -config string
        Configuration file path (default "config.yaml")
  -generate-config
        Generate example configuration file and exit
```

## Development

Build the server:

```bash
go build -o middleware-server
```

Run with custom config:

```bash
./middleware-server -config my-config.yaml
```

## Additional Examples

### Get Runtime Version

```bash
curl http://localhost:8080/api/v1/version
```

Response:

```json
{
  "success": true,
  "name": "Plc30 App",
  "major_version": 3,
  "minor_version": 1,
  "version_build": 1969,
  "version": "3.1.1969"
}
```

### Get PLC State

```bash
curl http://localhost:8080/api/v1/state
```

Response:

```json
{
  "success": true,
  "ads_state": 5,
  "ads_state_name": "Run",
  "device_state": 0
}
```

### Control PLC

Stop PLC:

```bash
curl -X POST http://localhost:8080/api/v1/control \
  -H "Content-Type: application/json" \
  -d '{"command": "stop"}'
```

Start PLC:

```bash
curl -X POST http://localhost:8080/api/v1/control \
  -H "Content-Type: application/json" \
  -d '{"command": "start"}'
```

Reset PLC:

```bash
curl -X POST http://localhost:8080/api/v1/control \
  -H "Content-Type: application/json" \
  -d '{"command": "reset"}'
```

Supported commands: `start`, `run`, `stop`, `reset`

## Coming Soon

- ⏳ Authentication & authorization
- ⏳ Rate limiting
- ⏳ Metrics endpoint
