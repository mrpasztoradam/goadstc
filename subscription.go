package goadstc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ads"
)

// Notification represents a single notification event from the PLC.
type Notification struct {
	Data      []byte
	Timestamp time.Time
}

// Subscription represents an active ADS notification subscription.
type Subscription struct {
	handle   uint32
	client   *Client
	notifCh  chan Notification
	closed   bool
	closeMu  sync.Mutex
	closeErr error

	// Stored for re-establishment after reconnect
	opts NotificationOptions
}

// NotificationOptions configures a notification subscription.
type NotificationOptions struct {
	IndexGroup       uint32
	IndexOffset      uint32
	Length           uint32
	TransmissionMode ads.TransmissionMode
	MaxDelay         time.Duration // Maximum delay before notification is sent
	CycleTime        time.Duration // Cycle time for cyclic notifications
}

// SymbolNotificationOptions configures a symbol-based notification subscription.
type SymbolNotificationOptions struct {
	TransmissionMode ads.TransmissionMode
	MaxDelay         time.Duration // Maximum delay before notification is sent
	CycleTime        time.Duration // Cycle time for cyclic notifications
}

// Notifications returns the channel for receiving notifications.
// The channel is closed when the subscription is closed.
func (s *Subscription) Notifications() <-chan Notification {
	return s.notifCh
}

// Handle returns the notification handle assigned by the PLC.
func (s *Subscription) Handle() uint32 {
	return s.handle
}

// Close unsubscribes from the notification and closes the notification channel.
// It's safe to call Close multiple times.
func (s *Subscription) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return s.closeErr
	}

	s.closed = true

	// Remove from client's registry
	s.client.unregisterSubscription(s.handle)

	// Delete notification on PLC
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := ads.DeleteDeviceNotificationRequest{
		NotificationHandle: s.handle,
	}

	reqData, err := req.MarshalBinary()
	if err != nil {
		s.closeErr = fmt.Errorf("marshal delete notification request: %w", err)
		close(s.notifCh)
		return s.closeErr
	}

	respPacket, err := s.client.sendRequest(ctx, ads.CmdDelDeviceNotification, reqData)
	if err != nil {
		s.closeErr = fmt.Errorf("delete notification: %w", err)
		close(s.notifCh)
		return s.closeErr
	}

	var resp ads.DeleteDeviceNotificationResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		s.closeErr = fmt.Errorf("unmarshal delete notification response: %w", err)
		close(s.notifCh)
		return s.closeErr
	}

	if resp.Result != 0 {
		s.closeErr = ads.Error(resp.Result)
		close(s.notifCh)
		return s.closeErr
	}

	close(s.notifCh)
	return nil
}

// notify is called internally to send a notification to the subscription.
func (s *Subscription) notify(data []byte, timestamp time.Time) {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return
	}

	// Non-blocking send to prevent deadlock
	select {
	case s.notifCh <- Notification{Data: data, Timestamp: timestamp}:
	default:
		// Channel full, drop notification
	}
}
