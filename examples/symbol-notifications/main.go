package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Symbol-Based Notifications Example (Milestone 6)     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	plcIP := "10.10.0.3:48898"
	plcNetID := ams.NetID{10, 0, 10, 20, 1, 1}
	pcNetID := ams.NetID{10, 10, 0, 10, 1, 1}

	fmt.Printf("🔌 Connecting to PLC at %s...\n", plcIP)
	client, err := goadstc.New(
		goadstc.WithTarget(plcIP),
		goadstc.WithAMSNetID(plcNetID),
		goadstc.WithSourceNetID(pcNetID),
		goadstc.WithAMSPort(851),
		goadstc.WithTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}
	defer client.Close()
	fmt.Println("✅ Connected successfully\n")

	ctx := context.Background()

	// Test 1: Subscribe using symbol name - much simpler than raw index group/offset!
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📡 Test 1: Subscribe to Variable by Symbol Name")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("   Subscribing to MAIN.uUint with OnChange mode...")

	sub, err := client.SubscribeSymbol(ctx, "MAIN.uUint", goadstc.SymbolNotificationOptions{
		TransmissionMode: ads.TransModeOnChange,
		MaxDelay:         100 * time.Millisecond,
		CycleTime:        50 * time.Millisecond,
	})
	if err != nil {
		log.Fatalf("❌ Failed to subscribe: %v", err)
	}
	defer sub.Close()
	
	fmt.Printf("✅ Subscribed successfully (handle: %d)\n", sub.Handle())
	fmt.Println("   Waiting for notifications... (try changing the value in TwinCAT)")
	fmt.Println()

	// Monitor notifications
	notifCount := 0
	done := make(chan bool)
	
	go func() {
		for notif := range sub.Notifications() {
			notifCount++
			if len(notif.Data) >= 2 {
				value := binary.LittleEndian.Uint16(notif.Data)
				fmt.Printf("   📬 [%d] Value changed: %d (at %s)\n",
					notifCount, value, notif.Timestamp.Format("15:04:05.000"))
			}
			
			if notifCount >= 3 {
				fmt.Println("\n   (Received 3 notifications, stopping...)")
				done <- true
				return
			}
		}
	}()

	// Wait for notifications or timeout
	select {
	case <-done:
		// Got 3 notifications
	case <-time.After(10 * time.Second):
		fmt.Println("\n   (Timeout after 10 seconds, no value changes detected)")
	}

	fmt.Println("\n✅ Test 1 complete")
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 6 Complete!                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("✨ Symbol-Based Notifications Key Benefits:")
	fmt.Println()
	fmt.Println("  Old way (manual):")
	fmt.Println("    1. Look up symbol to get IndexGroup/IndexOffset/Length")
	fmt.Println("    2. Create NotificationOptions with those values")
	fmt.Println("    3. Call Subscribe()")
	fmt.Println()
	fmt.Println("  New way (automatic):")
	fmt.Println("    client.SubscribeSymbol(ctx, \"MAIN.counter\", opts)")
	fmt.Println("    // Done! Just use the symbol name!")
	fmt.Println()
	fmt.Println("  ✅ No manual symbol lookup needed")
	fmt.Println("  ✅ Cleaner, more readable code")
	fmt.Println("  ✅ Automatically loads symbol table")
	fmt.Println("  ✅ Works with all transmission modes")
}
