// Package transport implements TCP transport for AMS/ADS communication.
package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ams"
)

// NotificationHandler is called when a notification packet is received.
type NotificationHandler func(*ams.Packet)

type Conn struct {
	conn                net.Conn
	mu                  sync.Mutex
	closed              atomic.Bool
	timeout             time.Duration
	invokeID            atomic.Uint32
	responses           chan *pendingResponse
	pending             map[uint32]chan<- *ams.Packet
	pendingMu           sync.RWMutex
	notificationHandler NotificationHandler
	notifHandlerMu      sync.RWMutex
}

type pendingResponse struct {
	invokeID uint32
	packet   *ams.Packet
	err      error
}

func Dial(ctx context.Context, address string, timeout time.Duration) (*Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	netConn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("transport: dial %s: %w", address, err)
	}

	conn := &Conn{
		conn:      netConn,
		timeout:   timeout,
		responses: make(chan *pendingResponse, 16),
		pending:   make(map[uint32]chan<- *ams.Packet),
	}

	go conn.readLoop()
	go conn.dispatchLoop()

	return conn, nil
}

func (c *Conn) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	err := c.conn.Close()

	c.pendingMu.Lock()
	for _, ch := range c.pending {
		close(ch)
	}
	c.pending = nil
	c.pendingMu.Unlock()

	close(c.responses)
	return err
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
	if c.closed.Load() {
		return nil, fmt.Errorf("transport: connection closed")
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
			return nil, err
		}
	}

	c.mu.Lock()
	err := ams.WritePacket(c.conn, req)
	c.mu.Unlock()

	if err != nil {
		return nil, err
	}

	select {
	case resp := <-respCh:
		if resp == nil {
			return nil, fmt.Errorf("transport: connection closed")
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("transport: timeout")
	}
}

func (c *Conn) readLoop() {
	for {
		if c.closed.Load() {
			return
		}

		if c.timeout > 0 {
			if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout * 2)); err != nil {
				c.responses <- &pendingResponse{err: err}
				return
			}
		}

		packet, err := ams.ReadPacket(c.conn)
		if err != nil {
			if !c.closed.Load() {
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
			c.Close()
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
			}
		}
	}
}
