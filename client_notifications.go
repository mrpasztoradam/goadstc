package goadstc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

// Subscribe creates a new notification subscription.
// The returned Subscription will deliver notifications via its Notifications() channel.
// Call Close() on the Subscription when done to clean up resources.
func (c *Client) Subscribe(ctx context.Context, opts NotificationOptions) (*Subscription, error) {
	req := ads.AddDeviceNotificationRequest{
		IndexGroup:       opts.IndexGroup,
		IndexOffset:      opts.IndexOffset,
		Length:           opts.Length,
		TransmissionMode: opts.TransmissionMode,
		MaxDelay:         uint32(opts.MaxDelay / time.Millisecond),
		CycleTime:        uint32(opts.CycleTime / time.Millisecond),
	}
	reqData, _ := req.MarshalBinary()

	respPacket, err := c.sendRequest(ctx, ads.CmdAddDeviceNotification, reqData)
	if err != nil {
		return nil, err
	}

	var resp ads.AddDeviceNotificationResponse
	if err := resp.UnmarshalBinary(respPacket.Data); err != nil {
		return nil, err
	}

	if resp.Result != 0 {
		return nil, ads.Error(resp.Result)
	}

	// Create subscription
	sub := &Subscription{
		handle:  resp.NotificationHandle,
		client:  c,
		notifCh: make(chan Notification, 16),
		closed:  false,
		closeMu: sync.Mutex{},
	}

	// Register subscription
	c.subscriptionsMu.Lock()
	c.subscriptions[sub.handle] = sub
	c.subscriptionsMu.Unlock()

	return sub, nil
}

// SubscribeSymbol creates a notification subscription using a symbol name.
// This is a convenience method that automatically looks up the symbol's index group,
// offset, and length. The returned Subscription will deliver notifications via its
// Notifications() channel. Call Close() on the Subscription when done.
func (c *Client) SubscribeSymbol(ctx context.Context, symbolName string, opts SymbolNotificationOptions) (*Subscription, error) {
	// Ensure symbols are loaded
	if err := c.ensureSymbolsLoaded(ctx); err != nil {
		return nil, fmt.Errorf("load symbols: %w", err)
	}

	// Get the symbol
	symbol, err := c.symbolTable.Get(symbolName)
	if err != nil {
		return nil, fmt.Errorf("get symbol %q: %w", symbolName, err)
	}

	// Create notification options with symbol information
	notifOpts := NotificationOptions{
		IndexGroup:       symbol.IndexGroup,
		IndexOffset:      symbol.IndexOffset,
		Length:           symbol.Size,
		TransmissionMode: opts.TransmissionMode,
		MaxDelay:         opts.MaxDelay,
		CycleTime:        opts.CycleTime,
	}

	return c.Subscribe(ctx, notifOpts)
}

// unregisterSubscription removes a subscription from the registry.
func (c *Client) unregisterSubscription(handle uint32) {
	c.subscriptionsMu.Lock()
	delete(c.subscriptions, handle)
	c.subscriptionsMu.Unlock()
}

// handleNotification processes incoming notification packets and routes them to subscriptions.
func (c *Client) handleNotification(packet *ams.Packet) {
	var notifReq ads.DeviceNotificationRequest
	if err := notifReq.UnmarshalBinary(packet.Data); err != nil {
		return
	}

	// Process each stamp in the notification
	for _, stamp := range notifReq.StampHeaders {
		// Convert Windows FILETIME to time.Time
		// FILETIME is 100-nanosecond intervals since 1601-01-01 00:00:00 UTC
		const fileTimeEpoch = 116444736000000000 // 100ns intervals between 1601 and 1970
		unixNano := int64(stamp.Timestamp-fileTimeEpoch) * 100
		timestamp := time.Unix(0, unixNano)

		for _, sample := range stamp.Samples {
			c.subscriptionsMu.RLock()
			sub, exists := c.subscriptions[sample.NotificationHandle]
			c.subscriptionsMu.RUnlock()

			if exists {
				sub.notify(sample.Data, timestamp)
			}
		}
	}
}
