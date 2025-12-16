# GoADS HTTP/WebSocket Middleware Architecture

## Overview

Full-featured HTTP REST API and WebSocket streaming interface for TwinCAT ADS operations with automatic JSON marshaling and Swagger documentation.

## Architecture Components

### 1. Package Structure

```
goadstc/
├── middleware/
│   ├── server.go          # HTTP server & routing
│   ├── handlers.go        # REST endpoint handlers
│   ├── websocket.go       # WebSocket connection manager
│   ├── types.go           # Request/Response types
│   ├── middleware.go      # JSON conversion layer
│   ├── swagger.go         # Swagger doc generation
│   └── errors.go          # Error handling & responses
└── examples/
    └── middleware-server/
        ├── main.go
        └── config.yaml
```

### 2. REST API Endpoints

#### **Symbol Operations**

```
GET    /api/v1/symbols                    # List all symbols
GET    /api/v1/symbols/{name}             # Get symbol metadata
GET    /api/v1/symbols/{name}/value       # Read symbol value
POST   /api/v1/symbols/{name}/value       # Write symbol value
POST   /api/v1/symbols/read               # Batch read
POST   /api/v1/symbols/write              # Batch write
```

#### **Struct Operations**

```
GET    /api/v1/structs/{name}             # Read entire struct
POST   /api/v1/structs/{name}/fields      # Write struct fields
GET    /api/v1/structs/{name}/metadata    # Get struct type info
```

#### **Subscription Management**

```
POST   /api/v1/subscriptions              # Create subscription (returns sub ID)
GET    /api/v1/subscriptions              # List active subscriptions
GET    /api/v1/subscriptions/{id}         # Get subscription details
DELETE /api/v1/subscriptions/{id}         # Cancel subscription
WebSocket: /ws/subscribe                   # WebSocket endpoint
```

#### **Type & Metadata**

```
GET    /api/v1/types                      # List all data types
GET    /api/v1/types/{name}               # Get type definition
```

#### **System Operations**

```
GET    /api/v1/health                     # Health check
GET    /api/v1/info                       # PLC connection info
GET    /api/v1/swagger                    # Swagger JSON
GET    /swagger-ui                        # Swagger UI
```

### 3. Request/Response Types

#### **Read Operations**

```json
// POST /api/v1/symbols/read
{
  "symbols": ["MAIN.temperature", "MAIN.counter"]
}

// Response
{
  "success": true,
  "data": {
    "MAIN.temperature": 25.5,
    "MAIN.counter": 42
  },
  "errors": {}
}
```

#### **Write Operations**

```json
// POST /api/v1/symbols/write
{
  "writes": {
    "MAIN.temperature": 30.0,
    "MAIN.enabled": true
  }
}

// Response
{
  "success": true,
  "results": {
    "MAIN.temperature": true,
    "MAIN.enabled": true
  },
  "errors": {}
}
```

#### **Struct Field Write**

```json
// POST /api/v1/structs/MAIN.myStruct/fields
{
  "fields": {
    "temperature": 25.5,
    "enabled": true,
    "counter": 100
  }
}

// Response
{
  "success": true,
  "fields_written": 3
}
```

#### **Subscription Creation**

```json
// POST /api/v1/subscriptions
{
  "symbol": "MAIN.temperature",
  "mode": "onchange",
  "cycle_time_ms": 100,
  "max_delay_ms": 500
}

// Response
{
  "subscription_id": "sub_1234",
  "symbol": "MAIN.temperature",
  "websocket_url": "ws://localhost:8080/ws/subscribe?id=sub_1234"
}
```

### 4. WebSocket Protocol

#### **Connection**

```
ws://localhost:8080/ws/subscribe?id=sub_1234
```

#### **Message Format**

```json
// Server → Client (Updates)
{
  "type": "update",
  "subscription_id": "sub_1234",
  "symbol": "MAIN.temperature",
  "value": 25.5,
  "timestamp": "2025-12-16T10:30:45.123Z"
}

// Server → Client (Error)
{
  "type": "error",
  "subscription_id": "sub_1234",
  "error": "Symbol not found"
}

// Client → Server (Subscribe to existing)
{
  "action": "subscribe",
  "subscription_id": "sub_1234"
}

// Client → Server (Ping/Pong)
{
  "action": "ping"
}
```

#### **Multiple Subscriptions per WebSocket**

Client can subscribe to multiple symbols on one WebSocket connection.

### 5. Swagger Integration

**Using:** `github.com/swaggo/swag` + `github.com/swaggo/http-swagger`

**Annotations in code:**

```go
// @Summary Read symbol value
// @Description Read the current value of a PLC symbol
// @Tags symbols
// @Accept json
// @Produce json
// @Param name path string true "Symbol name"
// @Success 200 {object} SymbolValueResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/symbols/{name}/value [get]
func (s *Server) HandleReadSymbol(w http.ResponseWriter, r *http.Request) {
```

**Auto-generate:** `swag init -g middleware/server.go`

### 6. Server Configuration

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  cors:
    enabled: true
    origins: ["*"]

plc:
  target: "10.10.0.3:48898"
  ams_net_id: "10.0.10.20.1.1"
  source_net_id: "10.10.0.10.1.1"
  ams_port: 851
  timeout_seconds: 5

middleware:
  max_batch_size: 100
  max_subscriptions: 1000
  websocket_buffer_size: 256

logging:
  level: "info" # debug, info, warn, error
  format: "json" # json, text
```

### 7. Technology Stack

**HTTP Framework:** `net/http` (stdlib) or `chi` or `gin` (lightweight routers)
**WebSocket:** `github.com/gorilla/websocket`
**Swagger:** `github.com/swaggo/swag`
**Config:** `gopkg.in/yaml.v3` or `github.com/spf13/viper`
**Optional Rate Limiting:** `golang.org/x/time/rate`

### 8. Error Response Format

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

**Error Codes:**

- `SYMBOL_NOT_FOUND`
- `INVALID_REQUEST`
- `TYPE_MISMATCH`
- `WRITE_FAILED`
- `SUBSCRIPTION_LIMIT_REACHED`
- `PLC_CONNECTION_ERROR`
- `INTERNAL_ERROR`

### 9. Type Conversion Rules

**Go Type → JSON Type:**

- `int8..int64, uint8..uint64` → `number`
- `float32, float64` → `number`
- `bool` → `boolean`
- `string` → `string`
- `time.Time` → `string` (RFC3339)
- `time.Duration` → `number` (milliseconds)
- `map[string]interface{}` (structs) → `object`
- `[]interface{}` (arrays) → `array`

**JSON Type → Go Type (writing):**

- Automatic type detection from symbol metadata
- Type validation before write
- Clear error messages for type mismatches

### 10. Authentication & Security (Future)

**Phase 1:** No auth (trusted network)
**Phase 2:** API key authentication
**Phase 3:** JWT tokens
**Phase 4:** TLS/HTTPS support

### 11. Metrics & Monitoring (Future)

**Prometheus metrics:**

- Request count by endpoint
- Request duration
- Active subscriptions
- WebSocket connections
- Error rates
- PLC connection status

## Implementation Phases

### **Phase 1: Core HTTP API** (Priority 1)

- [ ] Server setup with routing
- [ ] Basic CRUD endpoints for symbols
- [ ] Error handling
- [ ] JSON marshaling/unmarshaling
- [ ] Config file support

### **Phase 2: Swagger Documentation** (Priority 1)

- [ ] Swagger annotations on all endpoints
- [ ] Auto-generate swagger.json
- [ ] Serve Swagger UI at /swagger-ui
- [ ] Example requests/responses

### **Phase 3: Batch Operations** (Priority 2)

- [ ] Batch read endpoint
- [ ] Batch write endpoint
- [ ] Transaction-like behavior
- [ ] Performance optimization

### **Phase 4: Struct Operations** (Priority 2)

- [ ] Struct read endpoint
- [ ] Struct field write endpoint
- [ ] Type metadata endpoint

### **Phase 5: WebSocket Subscriptions** (Priority 1)

- [ ] WebSocket connection manager
- [ ] Subscription lifecycle management
- [ ] Multiple subscriptions per connection
- [ ] Graceful disconnect handling
- [ ] Reconnection support

### **Phase 6: Advanced Features** (Priority 3)

- [ ] CORS configuration
- [ ] Rate limiting
- [ ] Request logging
- [ ] Metrics endpoint
- [ ] Health checks with details

### **Phase 7: Example & Documentation** (Priority 1)

- [ ] Example server with config
- [ ] Postman collection
- [ ] Client examples (curl, JavaScript, Python)
- [ ] Performance benchmarks

## Testing Strategy

**Unit Tests:**

- Handler functions
- Type conversion
- Error handling

**Integration Tests:**

- Full HTTP request/response cycle
- WebSocket connections
- Subscription lifecycle

**Load Tests:**

- Concurrent requests
- Multiple WebSocket connections
- Batch operations

## Success Criteria

✅ Full REST API with all CRUD operations
✅ WebSocket streaming for subscriptions
✅ Complete Swagger documentation
✅ Configuration via YAML
✅ Comprehensive error handling
✅ Example server running and tested
✅ Client examples in multiple languages
