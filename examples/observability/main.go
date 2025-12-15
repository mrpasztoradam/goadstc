package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	// Create a structured logger (JSON format)
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := goadstc.NewSlogLogger(slog.New(logHandler))

	// Create an in-memory metrics collector
	metrics := goadstc.NewInMemoryMetrics()

	// Parse target address and NetID from environment
	target := os.Getenv("ADS_TARGET")
	if target == "" {
		target = "192.168.1.10:48898"
	}

	netID := ams.NetID{127, 0, 0, 1, 1, 1}
	if netIDStr := os.Getenv("ADS_NET_ID"); netIDStr != "" {
		// Parse NetID manually: "192.168.1.1.1.1" -> NetID{192, 168, 1, 1, 1, 1}
		fmt.Sscanf(netIDStr, "%d.%d.%d.%d.%d.%d",
			&netID[0], &netID[1], &netID[2], &netID[3], &netID[4], &netID[5])
	}

	// Create client with logging and metrics
	client, err := goadstc.New(
		goadstc.WithTarget(target),
		goadstc.WithAMSNetID(netID),
		goadstc.WithTimeout(5*time.Second),
		goadstc.WithAutoReconnect(true),
		goadstc.WithHealthCheck(10*time.Second),
		goadstc.WithLogger(logger),
		goadstc.WithMetrics(metrics),
		goadstc.WithStateCallback(func(oldState, newState goadstc.ConnectionState, err error) {
			fmt.Printf("Connection state changed: %s -> %s (error: %v)\n", oldState, newState, err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Perform some operations
	ctx := context.Background()

	// Read device state
	state, err := client.ReadState(ctx)
	if err != nil {
		if ce, ok := err.(*goadstc.ClassifiedError); ok {
			fmt.Printf("Read state failed: category=%s, retryable=%v, error=%v\n",
				ce.Category, ce.IsRetryable(), ce.Err)
		} else {
			fmt.Printf("Read state failed: %v\n", err)
		}
	} else {
		fmt.Printf("Device state: ADS=%d, Device=%d\n", state.ADSState, state.DeviceState)
	}

	// Read device info
	info, err := client.ReadDeviceInfo(ctx)
	if err != nil {
		if ce, ok := err.(*goadstc.ClassifiedError); ok {
			fmt.Printf("Read device info failed: category=%s, error=%v\n", ce.Category, ce.Err)
		}
	} else {
		fmt.Printf("Device: %s v%d.%d.%d\n", info.Name, info.MajorVersion, info.MinorVersion, info.VersionBuild)
	}

	// Try reading a variable (may fail if it doesn't exist)
	data, err := client.ReadSymbol(ctx, "MAIN.Counter")
	if err != nil {
		fmt.Printf("Read symbol failed (this is expected if symbol doesn't exist): %v\n", err)
	} else {
		fmt.Printf("Read symbol data: %d bytes\n", len(data))
	}

	// Print metrics snapshot
	time.Sleep(1 * time.Second) // Give time for async operations
	snapshot := metrics.Snapshot()

	fmt.Println("\n=== Metrics Summary ===")
	fmt.Printf("Connection Attempts:  %d\n", snapshot.ConnectionAttempts)
	fmt.Printf("Connection Successes: %d\n", snapshot.ConnectionSuccesses)
	fmt.Printf("Connection Failures:  %d\n", snapshot.ConnectionFailures)
	fmt.Printf("Connection Active:    %v\n", snapshot.ConnectionActive)
	fmt.Printf("Reconnections:        %d\n", snapshot.Reconnections)
	fmt.Printf("\n")

	fmt.Println("Operations:")
	for op, count := range snapshot.OperationCounts {
		errors := int64(0)
		if errCount, ok := snapshot.OperationErrors[op]; ok {
			errors = errCount
		}
		fmt.Printf("  %s: %d operations, %d errors\n", op, count, errors)
	}
	fmt.Printf("\n")

	fmt.Printf("Data Transfer:\n")
	fmt.Printf("  Bytes Sent:     %d\n", snapshot.BytesSent)
	fmt.Printf("  Bytes Received: %d\n", snapshot.BytesReceived)
	fmt.Printf("\n")

	if len(snapshot.ErrorsByCategory) > 0 {
		fmt.Println("Errors by Category:")
		for category, count := range snapshot.ErrorsByCategory {
			fmt.Printf("  %s: %d\n", category, count)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("Health Checks:\n")
	fmt.Printf("  Started:  %d\n", snapshot.HealthChecksStarted)
	fmt.Printf("  Success:  %d\n", snapshot.HealthChecksSuccess)
	fmt.Printf("  Failures: %d\n", snapshot.HealthChecksFailure)
}
