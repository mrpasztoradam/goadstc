package transport

import (
	"context"
	"testing"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func TestConnectionState(t *testing.T) {
	conn := &Conn{}
	conn.state.Store(int32(StateConnecting))
	
	if state := conn.getState(); state != StateConnecting {
		t.Errorf("Expected StateConnecting, got %v", state)
	}

	conn.state.Store(int32(StateConnected))
	if state := conn.getState(); state != StateConnected {
		t.Errorf("Expected StateConnected, got %v", state)
	}
}

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
		{StateDisconnecting, "disconnecting"},
		{StateClosed, "closed"},
		{StateError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestCompareAndSwapState(t *testing.T) {
	conn := &Conn{}
	conn.state.Store(int32(StateConnected))

	// Should succeed
	if !conn.compareAndSwapState(StateConnected, StateDisconnecting) {
		t.Error("Expected CAS to succeed")
	}

	// Should fail - state is now Disconnecting
	if conn.compareAndSwapState(StateConnected, StateClosed) {
		t.Error("Expected CAS to fail")
	}

	if state := conn.getState(); state != StateDisconnecting {
		t.Errorf("Expected StateDisconnecting, got %v", state)
	}
}

func TestErrorHandling(t *testing.T) {
	conn := &Conn{}
	
	// No error initially
	if err := conn.getError(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Set an error
	testErr := ErrConnectionFailed
	conn.setError(testErr)

	if err := conn.getError(); err != testErr {
		t.Errorf("Expected %v, got %v", testErr, err)
	}

	if state := conn.getState(); state != StateError {
		t.Errorf("Expected StateError, got %v", state)
	}
}

func TestGracefulShutdown(t *testing.T) {
	// This is a unit test for the shutdown mechanism without actual network
	ctx, cancel := context.WithCancel(context.Background())
	
	conn := &Conn{
		pending:        make(map[uint32]chan<- *ams.Packet),
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
	conn.state.Store(int32(StateConnected))

	// Add some pending requests
	ch1 := make(chan *ams.Packet, 1)
	ch2 := make(chan *ams.Packet, 1)
	conn.pending[1] = ch1
	conn.pending[2] = ch2

	// Shutdown should clear pending
	go func() {
		time.Sleep(100 * time.Millisecond)
		conn.state.Store(int32(StateDisconnecting))
		conn.shutdownCancel()
		
		conn.pendingMu.Lock()
		for _, ch := range conn.pending {
			close(ch)
		}
		conn.pending = nil
		conn.pendingMu.Unlock()
	}()

	// Wait for shutdown
	<-conn.shutdownCtx.Done()

	conn.pendingMu.Lock()
	pendingCount := len(conn.pending)
	conn.pendingMu.Unlock()

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending requests, got %d", pendingCount)
	}
}
