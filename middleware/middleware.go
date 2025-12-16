package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/symbols"
)

// Middleware provides JSON-based operations over a GoADS client
type Middleware struct {
	client        *goadstc.Client
	subscriptions map[string]*goadstc.Subscription
	subMutex      sync.RWMutex
	subManager    *SubscriptionManager
	config        *Config
	startTime     time.Time
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(client *goadstc.Client, config *Config) *Middleware {
	return &Middleware{
		client:        client,
		subscriptions: make(map[string]*goadstc.Subscription),
		subManager:    NewSubscriptionManager(client, config.Middleware.MaxSubscriptions),
		config:        config,
		startTime:     time.Now(),
	}
}

// ReadSymbol reads a single symbol value
func (m *Middleware) ReadSymbol(ctx context.Context, symbolName string) (*SymbolValueResponse, error) {
	value, err := m.client.ReadSymbolValue(ctx, symbolName)
	if err != nil {
		return &SymbolValueResponse{
			Success: false,
			Symbol:  symbolName,
			Error:   err.Error(),
		}, nil
	}

	return &SymbolValueResponse{
		Success: true,
		Symbol:  symbolName,
		Value:   value,
		Type:    fmt.Sprintf("%T", value),
	}, nil
}

// BatchRead reads multiple symbols
func (m *Middleware) BatchRead(ctx context.Context, symbolNames []string) (*BatchReadResponse, error) {
	if len(symbolNames) > m.config.Middleware.MaxBatchSize {
		return nil, NewBatchSizeExceededError(len(symbolNames), m.config.Middleware.MaxBatchSize)
	}

	data := make(map[string]interface{})
	errors := make(map[string]string)

	for _, symbolName := range symbolNames {
		value, err := m.client.ReadSymbolValue(ctx, symbolName)
		if err != nil {
			errors[symbolName] = err.Error()
		} else {
			data[symbolName] = value
		}
	}

	return &BatchReadResponse{
		Success: len(errors) == 0,
		Data:    data,
		Errors:  errors,
	}, nil
}

// WriteSymbol writes a single symbol value
func (m *Middleware) WriteSymbol(ctx context.Context, symbolName string, value interface{}) (*WriteSymbolResponse, error) {
	err := m.client.WriteSymbolValue(ctx, symbolName, value)
	if err != nil {
		return &WriteSymbolResponse{
			Success: false,
			Symbol:  symbolName,
			Error:   err.Error(),
		}, nil
	}

	return &WriteSymbolResponse{
		Success: true,
		Symbol:  symbolName,
	}, nil
}

// BatchWrite writes multiple symbols
func (m *Middleware) BatchWrite(ctx context.Context, writes map[string]interface{}) (*BatchWriteResponse, error) {
	if len(writes) > m.config.Middleware.MaxBatchSize {
		return nil, NewBatchSizeExceededError(len(writes), m.config.Middleware.MaxBatchSize)
	}

	results := make(map[string]bool)
	errors := make(map[string]string)

	for symbolName, value := range writes {
		err := m.client.WriteSymbolValue(ctx, symbolName, value)
		if err != nil {
			results[symbolName] = false
			errors[symbolName] = err.Error()
		} else {
			results[symbolName] = true
		}
	}

	return &BatchWriteResponse{
		Success: len(errors) == 0,
		Results: results,
		Errors:  errors,
	}, nil
}

// WriteStructFields writes struct fields using byte offset method
func (m *Middleware) WriteStructFields(ctx context.Context, symbolName string, fields map[string]interface{}) (*WriteStructFieldsResponse, error) {
	err := m.client.WriteStructFields(ctx, symbolName, fields)
	if err != nil {
		return &WriteStructFieldsResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &WriteStructFieldsResponse{
		Success:       true,
		FieldsWritten: len(fields),
	}, nil
}

// GetSymbolTable retrieves all symbols from the PLC
func (m *Middleware) GetSymbolTable(ctx context.Context) (*SymbolTableResponse, error) {
	symbolsList, err := m.client.ListSymbols(ctx)
	if err != nil {
		return &SymbolTableResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	symbols := make([]SymbolInfo, len(symbolsList))
	for i, sym := range symbolsList {
		symbols[i] = SymbolInfo{
			Name:        sym.Name,
			Type:        sym.Type.Name,
			Size:        sym.Size,
			IndexGroup:  sym.IndexGroup,
			IndexOffset: sym.IndexOffset,
			Comment:     sym.Comment,
		}
	}

	return &SymbolTableResponse{
		Success: true,
		Count:   len(symbols),
		Symbols: symbols,
	}, nil
}

// GetSymbolInfo retrieves metadata for a specific symbol
func (m *Middleware) GetSymbolInfo(ctx context.Context, symbolName string) (*SymbolInfo, error) {
	// Get the symbol from the table
	symbolsList, err := m.client.ListSymbols(ctx)
	if err != nil {
		return nil, NewInternalError(fmt.Sprintf("failed to list symbols: %v", err))
	}

	for _, sym := range symbolsList {
		if sym.Name == symbolName {
			return &SymbolInfo{
				Name:        sym.Name,
				Type:        sym.Type.Name,
				Size:        sym.Size,
				IndexGroup:  sym.IndexGroup,
				IndexOffset: sym.IndexOffset,
				Comment:     sym.Comment,
			}, nil
		}
	}

	return nil, NewSymbolNotFoundError(symbolName)
}

// GetHealth returns the health status
func (m *Middleware) GetHealth() *HealthResponse {
	// TODO: Add actual connection check
	return &HealthResponse{
		Status:    "ok",
		Connected: true,
		Timestamp: time.Now(),
	}
}

// GetInfo returns server and PLC connection information
func (m *Middleware) GetInfo(ctx context.Context) (*InfoResponse, error) {
	symbolsList, err := m.client.ListSymbols(ctx)
	symbolCount := 0
	if err == nil {
		symbolCount = len(symbolsList)
	}

	return &InfoResponse{
		Target:       m.config.PLC.Target,
		AMSNetID:     m.config.PLC.AMSNetID,
		SourceNetID:  m.config.PLC.SourceNetID,
		AMSPort:      m.config.PLC.AMSPort,
		Connected:    err == nil,
		SymbolCount:  symbolCount,
		ServerUptime: time.Since(m.startTime).String(),
	}, nil
}

// GetVersion retrieves the runtime version information
func (m *Middleware) GetVersion(ctx context.Context) *VersionResponse {
	info, err := m.client.ReadDeviceInfo(ctx)
	if err != nil {
		return &VersionResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	version := fmt.Sprintf("%d.%d.%d", info.MajorVersion, info.MinorVersion, info.VersionBuild)
	return &VersionResponse{
		Success:      true,
		Name:         info.Name,
		MajorVersion: info.MajorVersion,
		MinorVersion: info.MinorVersion,
		VersionBuild: info.VersionBuild,
		Version:      version,
	}
}

// GetState retrieves the current PLC state
func (m *Middleware) GetState(ctx context.Context) *StateResponse {
	state, err := m.client.ReadState(ctx)
	if err != nil {
		return &StateResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &StateResponse{
		Success:      true,
		ADSState:     uint16(state.ADSState),
		ADSStateName: state.ADSState.String(),
		DeviceState:  state.DeviceState,
	}
}

// Control executes a PLC control command (start, stop, reset)
func (m *Middleware) Control(ctx context.Context, command string) *ControlResponse {
	var adsState ads.ADSState

	switch command {
	case "start", "run":
		adsState = ads.StateRun
	case "stop":
		adsState = ads.StateStop
	case "reset":
		adsState = ads.StateReset
	default:
		return &ControlResponse{
			Success: false,
			Command: command,
			Error:   fmt.Sprintf("unknown command: %s (supported: start, stop, reset)", command),
		}
	}

	err := m.client.WriteControl(ctx, adsState, 0, nil)
	if err != nil {
		return &ControlResponse{
			Success: false,
			Command: command,
			Error:   err.Error(),
		}
	}

	return &ControlResponse{
		Success: true,
		Command: command,
	}
}

// Helper function to convert symbols.Symbol to SymbolInfo
func symbolToInfo(sym *symbols.Symbol) SymbolInfo {
	return SymbolInfo{
		Name:        sym.Name,
		Type:        sym.Type.Name,
		Size:        sym.Size,
		IndexGroup:  sym.IndexGroup,
		IndexOffset: sym.IndexOffset,
		Comment:     sym.Comment,
	}
}
