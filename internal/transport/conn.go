// Package transport implements TCP transport for AMS/ADS communication.
package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ams"
)

// ConnectionState represents the state of the connection.
type ConnectionState int32

const (
	StateConnecting ConnectionState = iota
	StateConnected
	StateDisconnecting
	StateClosed
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDisconnecting:
		return "disconnecting"
	case StateClosed:
		return "closed"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrConnectionFailed = errors.New("connection failed")
)

// NotificationHandler is called when a notification packet is received.
type NotificationHandler func(*ams.Packet)

type Conn struct {
	conn                net.Conn
	mu                  sync.Mutex
	state               atomic.Int32 // ConnectionState
	timeout             time.Duration
	invokeID            atomic.Uint32
	responses           chan *pendingResponse
	pending             map[uint32]chan<- *ams.Packet
	pendingMu           sync.RWMutex
	notificationHandler NotificationHandler
	notifHandlerMu      sync.RWMutex
	shutdownCtx         context.Context
	shutdownCancel      context.CancelFunc
	lastError           error
	errorMu             sync.RWMutex
}

type pendingResponse struct {
	invokeID uint32
	packet   *ams.Packet
	err      error
}

func Dial(ctx context.Context, address string, timeout time.Duration) (*Conn, error) {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second, // Enable TCP keepalive
	}
	netConn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("transport: dial %s: %w", address, err)
	}

	// Configure TCP socket for better connection handling
	if tcpConn, ok := netConn.(*net.TCPConn); ok {
		// Enable TCP keepalive at OS level
		if err := tcpConn.SetKeepAlive(true); err != nil {
			netConn.Close()
			return nil, fmt.Errorf("transport: failed to set keepalive: %w", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			netConn.Close()
			return nil, fmt.Errorf("transport: failed to set keepalive period: %w", err)
		}
		// Disable Nagle's algorithm for lower latency
		if err := tcpConn.SetNoDelay(true); err != nil {
			netConn.Close()
			return nil, fmt.Errorf("transport: failed to set nodelay: %w", err)
		}
		// Enable address reuse to reduce TIME_WAIT issues
		if err := tcpConn.SetLinger(0); err != nil {
			netConn.Close()
			return nil, fmt.Errorf("transport: failed to set linger: %w", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	conn := &Conn{
		conn:           netConn,
		timeout:        timeout,
		responses:      make(chan *pendingResponse, 16),
		pending:        make(map[uint32]chan<- *ams.Packet),
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
	conn.state.Store(int32(StateConnected))

	go conn.readLoop()
	go conn.dispatchLoop()

	return conn, nil
}

func (c *Conn) Close() error {
	return c.CloseWithTimeout(5 * time.Second)
}

// CloseWithTimeout closes the connection with a timeout for graceful shutdown.
func (c *Conn) CloseWithTimeout(timeout time.Duration) error {
	// Transition to disconnecting state
	if !c.compareAndSwapState(StateConnected, StateDisconnecting) {
		currentState := ConnectionState(c.state.Load())
		if currentState == StateClosed || currentState == StateDisconnecting {
			return nil // Already closing or closed
		}
		c.state.Store(int32(StateDisconnecting))
	}

	// Signal shutdown to goroutines
	c.shutdownCancel()

	// Wait for pending operations with timeout
	done := make(chan struct{})
	go func() {
		c.pendingMu.Lock()
		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = nil
		c.pendingMu.Unlock()
		close(done)
	}()

	select {
	case <-done:
		// All pending operations cleared
	case <-time.After(timeout):
		// Timeout - force close
		c.setError(fmt.Errorf("close timeout: %d pending requests abandoned", len(c.pending)))
	}

	// Close the network connection
	err := c.conn.Close()

	// Close response channel (will terminate dispatchLoop)
	close(c.responses)

	// Mark as fully closed
	c.state.Store(int32(StateClosed))

	return err
}

func (c *Conn) compareAndSwapState(old, new ConnectionState) bool {
	return c.state.CompareAndSwap(int32(old), int32(new))
}

func (c *Conn) getState() ConnectionState {
	return ConnectionState(c.state.Load())
}

func (c *Conn) setError(err error) {
	c.errorMu.Lock()
	c.lastError = err
	c.errorMu.Unlock()
	c.state.Store(int32(StateError))
}

func (c *Conn) getError() error {
	c.errorMu.RLock()
	defer c.errorMu.RUnlock()
	return c.lastError
}

func (c *Conn) NextInvokeID() uint32 {
	return c.invokeID.Add(1)
}

// SetNotificationHandler sets the handler for notification packets (CommandID 0x0008).
func (c *Conn) SetNotificationHandler(handler NotificationHandler) {
	c.notifHandlerMu.Lock()
	c.notificationHandler = handler
	c.notifHandlerMu.Unlock()
}

func (c *Conn) SendRequest(ctx context.Context, req *ams.Packet) (*ams.Packet, error) {
	state := c.getState()
	if state != StateConnected {
		if err := c.getError(); err != nil {
			return nil, fmt.Errorf("transport: connection %s: %w", state, err)
		}
		return nil, fmt.Errorf("transport: connection %s", state)
	}

	respCh := make(chan *ams.Packet, 1)
	invokeID := req.Header.InvokeID

	c.pendingMu.Lock()
	c.pending[invokeID] = respCh
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, invokeID)
		c.pendingMu.Unlock()
	}()

	if c.timeout > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
			c.setError(err)
			return nil, fmt.Errorf("transport: failed to set write deadline: %w", err)
		}
	}

	c.mu.Lock()
	err := ams.WritePacket(c.conn, req)
	c.mu.Unlock()

	if err != nil {
		c.setError(err)
		return nil, fmt.Errorf("transport: write failed: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp == nil {
			if err := c.getError(); err != nil {
				return nil, fmt.Errorf("transport: connection closed: %w", err)
			}
			return nil, ErrConnectionClosed
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.shutdownCtx.Done():
		return nil, ErrConnectionClosed
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("transport: request timeout after %v", c.timeout)
	}
}

func (c *Conn) readLoop() {
	defer func() {
		// Ensure connection is marked as closed if readLoop exits
		if c.getState() == StateConnected {
			c.setError(errors.New("read loop terminated unexpectedly"))
		}
	}()

	for {
		select {
		case <-c.shutdownCtx.Done():
			return
		default:
		}

		if c.getState() != StateConnected {
			return
		}

		if c.timeout > 0 {
			if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout * 2)); err != nil {
				c.setError(fmt.Errorf("failed to set read deadline: %w", err))
				c.responses <- &pendingResponse{err: err}
				return
			}
		}

		packet, err := ams.ReadPacket(c.conn)
		if err != nil {
			if c.getState() == StateConnected {
				c.setError(fmt.Errorf("read packet failed: %w", err))
				c.responses <- &pendingResponse{err: err}
			}
			return
		}

		c.responses <- &pendingResponse{
			invokeID: packet.Header.InvokeID,
			packet:   packet,
		}
	}
}

func (c *Conn) dispatchLoop() {
	for resp := range c.responses {
		if resp.err != nil {
			// Error in read loop - initiate graceful close
			go c.Close()
			return
		}

		// Check if this is a notification packet (CommandID 0x0008)
		if resp.packet.Header.CommandID == 0x0008 {
			c.notifHandlerMu.RLock()
			handler := c.notificationHandler
			c.notifHandlerMu.RUnlock()

			if handler != nil {
				go handler(resp.packet)
			}
			continue
		}

		// Regular response packet - match by InvokeID
		c.pendingMu.RLock()
		ch, ok := c.pending[resp.invokeID]
		c.pendingMu.RUnlock()

		if ok && ch != nil {
			select {
			case ch <- resp.packet:
			default:
				// Channel full or closed - log but don't block
			}
		}
	}
}
