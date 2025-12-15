package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ads"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	plcIP := "10.10.0.3:48898"
	plcNetID := ams.NetID{10, 0, 10, 20, 1, 1}
	pcNetID := ams.NetID{10, 10, 0, 10, 1, 1}

	fmt.Printf("🔌 Connecting to PLC at %s...\n", plcIP)
	// Create ADS client
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
	fmt.Println("✅ Connected successfully")
	fmt.Println()

	ctx := context.Background()

	// Read current state
	fmt.Println("Reading current PLC state...")
	state, err := client.ReadState(ctx)
	if err != nil {
		log.Fatalf("Failed to read state: %v", err)
	}
	fmt.Printf("Current state - ADS: %d, Device: %d\n\n", state.ADSState, state.DeviceState)

	// Example 1: Stop PLC
	fmt.Println("Stopping PLC...")
	if err := client.WriteControl(ctx, ads.StateStop, 0, nil); err != nil {
		log.Printf("Failed to stop PLC: %v", err)
	} else {
		fmt.Println("PLC stopped successfully")
	}

	// Read state after stop
	state, err = client.ReadState(ctx)
	if err != nil {
		log.Printf("Failed to read state: %v", err)
	} else {
		fmt.Printf("State after stop - ADS: %d, Device: %d\n\n", state.ADSState, state.DeviceState)
	}

	// Example 2: Start PLC
	fmt.Println("Starting PLC...")
	if err := client.WriteControl(ctx, ads.StateRun, 0, nil); err != nil {
		log.Printf("Failed to start PLC: %v", err)
	} else {
		fmt.Println("PLC started successfully")
	}

	// Read state after start
	state, err = client.ReadState(ctx)
	if err != nil {
		log.Printf("Failed to read state: %v", err)
	} else {
		fmt.Printf("State after start - ADS: %d, Device: %d\n\n", state.ADSState, state.DeviceState)
	}

	// Example 3: Reset PLC
	fmt.Println("Resetting PLC...")
	if err := client.WriteControl(ctx, ads.StateReset, 0, nil); err != nil {
		log.Printf("Failed to reset PLC: %v", err)
	} else {
		fmt.Println("PLC reset successfully")
	}

	// Read final state
	fmt.Println("\nReading final PLC state...")
	state, err = client.ReadState(ctx)
	if err != nil {
		log.Fatalf("Failed to read final state: %v", err)
	}
	fmt.Printf("Final state - ADS: %d, Device: %d\n", state.ADSState, state.DeviceState)
}
