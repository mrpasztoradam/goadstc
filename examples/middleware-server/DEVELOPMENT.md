# GoADS Middleware - Developer Guide

## VS Code Tasks

Access tasks via `Cmd+Shift+P` → "Tasks: Run Task" or `Cmd+Shift+B` for build tasks.

### Available Tasks

#### Server Management

- **Middleware: Build Server** - Compile the middleware server
- **Middleware: Start Server** - Start the server (runs in background)
- **Middleware: Stop Server** - Stop the running server
- **Middleware: Rebuild & Restart Server** - Complete rebuild and restart cycle
- **Middleware: View Server Logs** - Tail the server log file

#### Documentation

- **Middleware: Update Swagger Docs** - Regenerate Swagger documentation from code annotations
- **Middleware: Update Swagger & Restart** - Update docs and restart server in one step
- **Middleware: Open Swagger UI** - Open Swagger UI in browser

#### Testing

- **Middleware: Test Health Endpoint** - Quick health check
- **Middleware: Test Version Endpoint** - Test version endpoint
- **Middleware: Test State Endpoint** - Test PLC state endpoint

#### Configuration

- **Middleware: Generate Config Example** - Generate example configuration file

## Quick Start

1. **First Time Setup**

   ```bash
   # Run task: "Middleware: Generate Config Example"
   # Edit config.yaml with your PLC settings
   ```

2. **Start Development**

   ```bash
   # Run task: "Middleware: Rebuild & Restart Server"
   ```

3. **After Code Changes**

   ```bash
   # Run task: "Middleware: Rebuild & Restart Server"
   ```

4. **After Swagger Annotation Changes**
   ```bash
   # Run task: "Middleware: Update Swagger & Restart"
   ```

## API Testing

### Using REST Client Extension

Open `api-tests.http` and click "Send Request" above any request.

### Using Swagger UI

```bash
# Run task: "Middleware: Open Swagger UI"
# Or visit: http://localhost:8080/swagger-ui/index.html
```

### Using curl

```bash
# Health check
curl http://localhost:8080/api/v1/health | jq

# Get version
curl http://localhost:8080/api/v1/version | jq

# Read symbol
curl http://localhost:8080/api/v1/symbols/MAIN.i/value | jq

# Control PLC
curl -X POST http://localhost:8080/api/v1/control \
  -H "Content-Type: application/json" \
  -d '{"command": "stop"}' | jq
```

## Development Workflow

### Adding New Endpoints

1. Add types to `middleware/types.go`
2. Add handler to `middleware/handlers.go` with Swagger annotations
3. Add route to `middleware/server.go`
4. Update Swagger docs: Run task "Middleware: Update Swagger Docs"
5. Restart server: Run task "Middleware: Rebuild & Restart Server"

### Swagger Annotations

Always include these annotations on handler functions:

```go
// @Summary Short description
// @Description Detailed description
// @Tags category
// @Accept json
// @Produce json
// @Param name path string true "Description"
// @Success 200 {object} ResponseType
// @Failure 400 {object} ErrorResponse
// @Router /endpoint [method]
```

**Important**: Router paths should be relative to `@BasePath /api/v1`

- ✅ Correct: `@Router /version [get]`
- ❌ Wrong: `@Router /api/v1/version [get]`

## Debugging

### View Server Logs

```bash
# Run task: "Middleware: View Server Logs"
# Or manually:
tail -f examples/middleware-server/server.log
```

### Check if Server is Running

```bash
lsof -i :8080
```

### Manual Server Start (Foreground)

```bash
cd examples/middleware-server
./middleware-server
```

## Configuration

Edit `examples/middleware-server/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

plc:
  target: "10.10.0.3:48898"
  ams_net_id: "10.0.10.20.1.1"
  source_net_id: "10.10.0.10.1.1"
  ams_port: 851
  timeout_seconds: 30

middleware:
  max_batch_size: 100
  max_subscriptions: 1000
```

## Useful Commands

```bash
# Install Swagger CLI
go install github.com/swaggo/swag/cmd/swag@latest

# Format Go code
go fmt ./...

# Run tests
go test ./...

# Build for production
cd examples/middleware-server
CGO_ENABLED=0 go build -ldflags="-s -w" -o middleware-server

# Check dependencies
go mod tidy
go mod verify
```

## Recommended VS Code Extensions

- **Go** (golang.go) - Go language support
- **Swagger Viewer** (swaggo.swag) - View Swagger docs
- **REST Client** (humao.rest-client) - Test API from .http files
- **YAML** (redhat.vscode-yaml) - YAML syntax support

Install all with: `Cmd+Shift+P` → "Extensions: Show Recommended Extensions"
