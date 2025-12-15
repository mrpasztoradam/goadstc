package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	goadstc "github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ads"
)

func main() {
	// Create ADS client
	client, err := goadstc.New(
		goadstc.WithTarget("192.168.1.100:48898"),
		goadstc.WithAMSNetID([6]byte{192, 168, 1, 100, 1, 1}),
		goadstc.WithAMSPort(851), // PLC Runtime 1
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Subscribe to PLC variable changes
	// This example monitors a DINT (32-bit integer) at a specific address
	sub, err := client.Subscribe(context.Background(), goadstc.NotificationOptions{
		IndexGroup:       ads.IndexGroupPLCMemory, // PLC memory area
		IndexOffset:      0x1000,                  // Example offset
		Length:           4,                       // DINT = 4 bytes
		TransmissionMode: ads.TransModeOnChange,   // Notify only on value change
		MaxDelay:         100 * time.Millisecond,  // Max delay before notification
		CycleTime:        50 * time.Millisecond,   // Check interval
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Close()

	fmt.Printf("Subscribed with handle %d\n", sub.Handle())
	fmt.Println("Monitoring PLC variable for changes...")
	fmt.Println("Press Ctrl+C to stop")

	// Process notifications
	for notif := range sub.Notifications() {
		// Parse the notification data (DINT in this example)
		if len(notif.Data) >= 4 {
			value := int32(binary.LittleEndian.Uint32(notif.Data[0:4]))
			fmt.Printf("[%s] Value changed: %d (0x%08X)\n",
				notif.Timestamp.Format("15:04:05.000"),
				value,
				uint32(value))
		} else {
			fmt.Printf("[%s] Received %d bytes: %x\n",
				notif.Timestamp.Format("15:04:05.000"),
				len(notif.Data),
				notif.Data)
		}
	}

	fmt.Println("Notification channel closed")
}
