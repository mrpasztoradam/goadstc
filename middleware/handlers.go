package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// @title GoADS HTTP/WebSocket Middleware API
// @version 1.0
// @description REST API for interacting with TwinCAT PLC via ADS protocol
// @description
// @description ## Features
// @description - Read and write PLC symbols with automatic type detection
// @description - Batch operations for multiple symbols
// @description - Struct field manipulation using byte offsets
// @description - Symbol table retrieval with metadata
// @description - WebSocket streaming for real-time symbol notifications (coming soon)
//
// @contact.name GoADS Middleware
// @contact.url https://github.com/yourusername/goadstc
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
//
// @tag.name symbols
// @tag.description Symbol read/write operations
// @tag.name structs
// @tag.description Struct field manipulation
// @tag.name health
// @tag.description Health and info endpoints

// Handler contains HTTP request handlers
type Handler struct {
	middleware *Middleware
	upgrader   *websocket.Upgrader
}

// NewHandler creates a new handler
func NewHandler(middleware *Middleware) *Handler {
	return &Handler{
		middleware: middleware,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now (configure CORS properly in production)
			},
		},
	}
}

// HandleReadSymbol handles GET /api/v1/symbols/{name}/value
// @Summary Read symbol value
// @Description Read the current value of a PLC symbol with automatic type detection
// @Tags symbols
// @Produce json
// @Param name path string true "Symbol name" example("MAIN.temperature")
// @Success 200 {object} SymbolValueResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /symbols/{name}/value [get]
func (h *Handler) HandleReadSymbol(w http.ResponseWriter, r *http.Request) {
	symbolName := chi.URLParam(r, "name")
	if symbolName == "" {
		WriteError(w, NewInvalidRequestError("symbol name is required"))
		return
	}

	result, err := h.middleware.ReadSymbol(r.Context(), symbolName)
	if err != nil {
		WriteError(w, err)
		return
	}

	if !result.Success {
		WriteError(w, NewSymbolNotFoundError(symbolName))
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleWriteSymbol handles POST /api/v1/symbols/{name}/value
// @Summary Write symbol value
// @Description Write a value to a PLC symbol with automatic type encoding
// @Tags symbols
// @Accept json
// @Produce json
// @Param name path string true "Symbol name" example("MAIN.temperature")
// @Param body body WriteSymbolRequest true "Value to write"
// @Success 200 {object} WriteSymbolResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/symbols/{name}/value [post]
func (h *Handler) HandleWriteSymbol(w http.ResponseWriter, r *http.Request) {
	symbolName := chi.URLParam(r, "name")
	if symbolName == "" {
		WriteError(w, NewInvalidRequestError("symbol name is required"))
		return
	}

	var req WriteSymbolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid JSON body"))
		return
	}

	result, err := h.middleware.WriteSymbol(r.Context(), symbolName, req.Value)
	if err != nil {
		WriteError(w, err)
		return
	}

	if !result.Success {
		WriteError(w, NewWriteFailedError(symbolName, result.Error))
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleBatchRead handles POST /api/v1/symbols/read
// @Summary Batch read symbols
// @Description Read multiple symbol values in a single request
// @Tags symbols
// @Accept json
// @Produce json
// @Param body body BatchReadRequest true "List of symbols to read"
// @Success 200 {object} BatchReadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /symbols/read [post]
func (h *Handler) HandleBatchRead(w http.ResponseWriter, r *http.Request) {
	var req BatchReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid JSON body"))
		return
	}

	if len(req.Symbols) == 0 {
		WriteError(w, NewInvalidRequestError("symbols array cannot be empty"))
		return
	}

	result, err := h.middleware.BatchRead(r.Context(), req.Symbols)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleBatchWrite handles POST /api/v1/symbols/write
// @Summary Batch write symbols
// @Description Write multiple symbol values in a single request
// @Tags symbols
// @Accept json
// @Produce json
// @Param body body BatchWriteRequest true "Map of symbols to values"
// @Success 200 {object} BatchWriteResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /symbols/write [post]
func (h *Handler) HandleBatchWrite(w http.ResponseWriter, r *http.Request) {
	var req BatchWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid JSON body"))
		return
	}

	if len(req.Writes) == 0 {
		WriteError(w, NewInvalidRequestError("writes map cannot be empty"))
		return
	}

	result, err := h.middleware.BatchWrite(r.Context(), req.Writes)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleGetSymbolTable handles GET /api/v1/symbols
// @Summary Get symbol table
// @Description Retrieve all symbols from the PLC symbol table
// @Tags symbols
// @Produce json
// @Success 200 {object} SymbolTableResponse
// @Failure 500 {object} ErrorResponse
// @Router /symbols [get]
func (h *Handler) HandleGetSymbolTable(w http.ResponseWriter, r *http.Request) {
	result, err := h.middleware.GetSymbolTable(r.Context())
	if err != nil {
		WriteError(w, err)
		return
	}

	if !result.Success {
		WriteError(w, NewInternalError(result.Error))
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleGetSymbolInfo handles GET /api/v1/symbols/{name}
// @Summary Get symbol metadata
// @Description Retrieve metadata for a specific symbol
// @Tags symbols
// @Produce json
// @Param name path string true "Symbol name" example("MAIN.temperature")
// @Success 200 {object} SymbolInfo
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /symbols/{name} [get]
func (h *Handler) HandleGetSymbolInfo(w http.ResponseWriter, r *http.Request) {
	symbolName := chi.URLParam(r, "name")
	if symbolName == "" {
		WriteError(w, NewInvalidRequestError("symbol name is required"))
		return
	}

	result, err := h.middleware.GetSymbolInfo(r.Context(), symbolName)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleReadStruct handles GET /api/v1/structs/{name}
// @Summary Read struct
// @Description Read an entire struct with all its fields
// @Tags structs
// @Produce json
// @Param name path string true "Struct symbol name" example("MAIN.myStruct")
// @Success 200 {object} SymbolValueResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /structs/{name} [get]
func (h *Handler) HandleReadStruct(w http.ResponseWriter, r *http.Request) {
	// Same as reading a regular symbol - auto-detection handles it
	h.HandleReadSymbol(w, r)
}

// HandleWriteStructFields handles POST /api/v1/structs/{name}/fields
// @Summary Write struct fields
// @Description Write multiple fields to a struct using byte offset method
// @Tags structs
// @Accept json
// @Produce json
// @Param name path string true "Struct symbol name" example("MAIN.myStruct")
// @Param body body WriteStructFieldsRequest true "Fields to write"
// @Success 200 {object} WriteStructFieldsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/structs/{name}/fields [post]
func (h *Handler) HandleWriteStructFields(w http.ResponseWriter, r *http.Request) {
	symbolName := chi.URLParam(r, "name")
	if symbolName == "" {
		WriteError(w, NewInvalidRequestError("symbol name is required"))
		return
	}

	var req WriteStructFieldsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid JSON body"))
		return
	}

	if len(req.Fields) == 0 {
		WriteError(w, NewInvalidRequestError("fields map cannot be empty"))
		return
	}

	result, err := h.middleware.WriteStructFields(r.Context(), symbolName, req.Fields)
	if err != nil {
		WriteError(w, err)
		return
	}

	if !result.Success {
		WriteError(w, NewWriteFailedError(symbolName, result.Error))
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// HandleHealth handles GET /api/v1/health
// @Summary Health check
// @Description Check if the server and PLC connection are healthy
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	result := h.middleware.GetHealth()
	WriteJSON(w, http.StatusOK, result)
}

// HandleInfo handles GET /api/v1/info
// @Summary Server info
// @Description Get server and PLC connection information
// @Tags system
// @Produce json
// @Success 200 {object} InfoResponse
// @Router /info [get]
func (h *Handler) HandleInfo(w http.ResponseWriter, r *http.Request) {
	result, err := h.middleware.GetInfo(r.Context())
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

// HandleWebSocket handles WebSocket connections for real-time subscriptions
// @Summary WebSocket subscription endpoint
// @Description Establish WebSocket connection for real-time symbol value updates
// @Tags websocket
// @Accept json
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Router /ws/subscribe [get]
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Handle WebSocket connection
	h.middleware.HandleWebSocket(conn)
}

// HandleGetVersion handles GET /api/v1/version
// @Summary Get runtime version
// @Description Retrieve PLC runtime name and version information
// @Tags system
// @Produce json
// @Success 200 {object} VersionResponse
// @Failure 500 {object} ErrorResponse
// @Router /version [get]
func (h *Handler) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	result := h.middleware.GetVersion(r.Context())
	if !result.Success {
		WriteError(w, NewInternalError(result.Error))
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

// HandleGetState handles GET /api/v1/state
// @Summary Get PLC state
// @Description Retrieve current PLC state (running, stopped, etc.)
// @Tags control
// @Produce json
// @Success 200 {object} StateResponse
// @Failure 500 {object} ErrorResponse
// @Router /state [get]
func (h *Handler) HandleGetState(w http.ResponseWriter, r *http.Request) {
	result := h.middleware.GetState(r.Context())
	if !result.Success {
		WriteError(w, NewInternalError(result.Error))
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

// HandleControl handles POST /api/v1/control
// @Summary Control PLC
// @Description Execute PLC control commands (start, stop, reset)
// @Tags control
// @Accept json
// @Produce json
// @Param request body ControlRequest true "Control command"
// @Success 200 {object} ControlResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /control [post]
func (h *Handler) HandleControl(w http.ResponseWriter, r *http.Request) {
	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewInvalidRequestError("invalid JSON body"))
		return
	}

	if req.Command == "" {
		WriteError(w, NewInvalidRequestError("command is required"))
		return
	}

	result := h.middleware.Control(r.Context(), req.Command)
	if !result.Success {
		WriteError(w, NewInternalError(result.Error))
		return
	}

	WriteJSON(w, http.StatusOK, result)
}
