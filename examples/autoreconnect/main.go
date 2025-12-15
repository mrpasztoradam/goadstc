package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
	"github.com/mrpasztoradam/goadstc/internal/transport"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║        Auto-Reconnection Example                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	plcIP := "10.10.0.3:48898"
	plcNetID := ams.NetID{10, 0, 10, 20, 1, 1}
	pcNetID := ams.NetID{10, 10, 0, 10, 1, 1}

	var connectionCount atomic.Int32

	// Create client with auto-reconnect enabled
	fmt.Println("Creating client with auto-reconnect...")
	client, err := goadstc.New(
		goadstc.WithTarget(plcIP),
		goadstc.WithAMSNetID(plcNetID),
		goadstc.WithSourceNetID(pcNetID),
		goadstc.WithAMSPort(851),
		goadstc.WithTimeout(5*time.Second),
		goadstc.WithAutoReconnect(true),
		goadstc.WithMaxReconnectDelay(30*time.Second),
		goadstc.WithHealthCheck(10*time.Second),
		goadstc.WithStateCallback(func(oldState, newState transport.ConnectionState, err error) {
			timestamp := time.Now().Format("15:04:05")
			switch newState {
			case transport.StateConnected:
				connectionCount.Add(1)
				fmt.Printf("[%s] ✅ Connected\n", timestamp)
			case transport.StateConnecting:
				fmt.Printf("[%s] 🔄 Reconnecting...\n", timestamp)
			case transport.StateError:
				fmt.Printf("[%s] ❌ Error: %v\n", timestamp, err)
			}
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test normal operations
	info, err := client.ReadDeviceInfo(ctx)
	if err == nil {
		fmt.Printf("Device: %s v%d.%d.%d\n", info.Name,
			info.MajorVersion, info.MinorVersion, info.VersionBuild)
	}

	// Continuous monitoring
	fmt.Println("\nMonitoring connection (Press Ctrl+C to stop)...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Printf("\nTotal connections: %d\n", connectionCount.Load())
			return
		case <-ticker.C:
			_, err := client.ReadState(ctx)
			if err != nil {
				fmt.Printf("[%s] ❌ ReadState failed: %v\n", time.Now().Format("15:04:05"), err)
			} else {
				fmt.Printf("[%s] ✅ ReadState OK\n", time.Now().Format("15:04:05"))
			}
		}
	}
}
