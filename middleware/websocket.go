package middleware

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mrpasztoradam/goadstc"
)

// SubscriptionManager manages WebSocket subscriptions
type SubscriptionManager struct {
	client        *goadstc.Client
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
	maxSubs       int
}

// Subscription represents an active WebSocket subscription
type Subscription struct {
	ID          string
	SymbolNames []string
	Interval    time.Duration
	Connection  *websocket.Conn
	cancelFunc  context.CancelFunc
	lastValues  map[string]interface{}
	mu          sync.RWMutex
}

// WebSocketMessage represents messages sent over WebSocket
type WebSocketMessage struct {
	Type      string                 `json:"type"` // "subscribe", "unsubscribe", "data", "error"
	RequestID string                 `json:"request_id,omitempty"`
	Symbols   []string               `json:"symbols,omitempty"`
	Interval  int                    `json:"interval,omitempty"` // milliseconds
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(client *goadstc.Client, maxSubscriptions int) *SubscriptionManager {
	return &SubscriptionManager{
		client:        client,
		subscriptions: make(map[string]*Subscription),
		maxSubs:       maxSubscriptions,
	}
}

// Subscribe creates a new subscription for the given symbols
func (sm *SubscriptionManager) Subscribe(conn *websocket.Conn, requestID string, symbolNames []string, interval time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check subscription limit
	if len(sm.subscriptions) >= sm.maxSubs {
		return NewInvalidRequestError("maximum subscription limit reached")
	}

	// Check if subscription already exists
	if _, exists := sm.subscriptions[requestID]; exists {
		return NewInvalidRequestError("subscription ID already exists")
	}

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create subscription
	sub := &Subscription{
		ID:          requestID,
		SymbolNames: symbolNames,
		Interval:    interval,
		Connection:  conn,
		cancelFunc:  cancel,
		lastValues:  make(map[string]interface{}),
	}

	sm.subscriptions[requestID] = sub

	// Start polling goroutine
	go sm.pollSymbols(ctx, sub)

	return nil
}

// Unsubscribe removes a subscription
func (sm *SubscriptionManager) Unsubscribe(requestID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[requestID]
	if !exists {
		return NewSymbolNotFoundError("subscription not found")
	}

	// Cancel the context to stop polling
	sub.cancelFunc()
	delete(sm.subscriptions, requestID)

	return nil
}

// UnsubscribeAll removes all subscriptions for a connection
func (sm *SubscriptionManager) UnsubscribeAll(conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, sub := range sm.subscriptions {
		if sub.Connection == conn {
			sub.cancelFunc()
			delete(sm.subscriptions, id)
		}
	}
}

// pollSymbols polls symbols and sends updates via WebSocket
func (sm *SubscriptionManager) pollSymbols(ctx context.Context, sub *Subscription) {
	ticker := time.NewTicker(sub.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.readAndSendSymbols(ctx, sub)
		}
	}
}

// readAndSendSymbols reads symbol values and sends them if changed
func (sm *SubscriptionManager) readAndSendSymbols(ctx context.Context, sub *Subscription) {
	data := make(map[string]interface{})
	changed := false

	sub.mu.RLock()
	symbols := sub.SymbolNames
	sub.mu.RUnlock()

	// Read all symbols
	for _, symbolName := range symbols {
		value, err := sm.client.ReadSymbolValue(ctx, symbolName)
		if err != nil {
			log.Printf("Error reading symbol %s: %v", symbolName, err)
			continue
		}

		sub.mu.Lock()
		lastValue, exists := sub.lastValues[symbolName]
		if !exists || lastValue != value {
			sub.lastValues[symbolName] = value
			changed = true
		}
		sub.mu.Unlock()

		data[symbolName] = value
	}

	// Only send if values changed
	if changed {
		msg := WebSocketMessage{
			Type:      "data",
			RequestID: sub.ID,
			Data:      data,
			Timestamp: time.Now(),
		}

		if err := sub.Connection.WriteJSON(msg); err != nil {
			log.Printf("Error sending WebSocket message: %v", err)
		}
	}
}

// GetSubscriptionCount returns the number of active subscriptions
func (sm *SubscriptionManager) GetSubscriptionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.subscriptions)
}

// HandleWebSocket handles WebSocket connections
func (m *Middleware) HandleWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	// Set up ping/pong handlers for connection keepalive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Ping goroutine
	go func() {
		for range pingTicker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	// Read messages
	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle message
		switch msg.Type {
		case "subscribe":
			if len(msg.Symbols) == 0 {
				m.sendWebSocketError(conn, msg.RequestID, "no symbols specified")
				continue
			}

			interval := time.Duration(msg.Interval) * time.Millisecond
			if interval == 0 {
				interval = 1000 * time.Millisecond // Default 1 second
			}

			if err := m.subManager.Subscribe(conn, msg.RequestID, msg.Symbols, interval); err != nil {
				m.sendWebSocketError(conn, msg.RequestID, err.Error())
			} else {
				response := WebSocketMessage{
					Type:      "subscribed",
					RequestID: msg.RequestID,
					Symbols:   msg.Symbols,
					Timestamp: time.Now(),
				}
				conn.WriteJSON(response)
			}

		case "unsubscribe":
			if err := m.subManager.Unsubscribe(msg.RequestID); err != nil {
				m.sendWebSocketError(conn, msg.RequestID, err.Error())
			} else {
				response := WebSocketMessage{
					Type:      "unsubscribed",
					RequestID: msg.RequestID,
					Timestamp: time.Now(),
				}
				conn.WriteJSON(response)
			}

		default:
			m.sendWebSocketError(conn, msg.RequestID, "unknown message type")
		}
	}

	// Clean up subscriptions when connection closes
	m.subManager.UnsubscribeAll(conn)
}

// sendWebSocketError sends an error message via WebSocket
func (m *Middleware) sendWebSocketError(conn *websocket.Conn, requestID, message string) {
	msg := WebSocketMessage{
		Type:      "error",
		RequestID: requestID,
		Error:     message,
		Timestamp: time.Now(),
	}
	conn.WriteJSON(msg)
}
