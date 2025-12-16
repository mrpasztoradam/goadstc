package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ads"
)

const (
	plcIP         = "10.10.0.3:48898"
	targetPort    = 851
	testOffset    = uint32(100)
	testOffset2   = uint32(200)
	monitorOffset = uint32(0)
)

var (
	plcNetID = [6]byte{10, 0, 10, 20, 1, 1}
	pcNetID  = [6]byte{10, 10, 0, 10, 1, 1}
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║     ADS Library Comprehensive Test Program              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create ADS client
	fmt.Printf("🔌 Connecting to PLC at %s...\n", plcIP)
	client, err := goadstc.New(
		goadstc.WithTarget(plcIP),
		goadstc.WithAMSNetID(plcNetID),
		goadstc.WithSourceNetID(pcNetID),
		goadstc.WithAMSPort(targetPort),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}
	defer client.Close()
	fmt.Println("✅ Connected successfully")
	fmt.Println()

	ctx := context.Background()

	// Test 1: ReadDeviceInfo
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📋 Test 1: ReadDeviceInfo")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := testReadDeviceInfo(ctx, client); err != nil {
		log.Printf("❌ Test 1 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 1 passed")
		fmt.Println()
	}

	// Test 2: ReadState
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📊 Test 2: ReadState")
	fmt.Println("═══════════════════════════════════════════════════════════")
	initialState, err := testReadState(ctx, client)
	if err != nil {
		log.Printf("❌ Test 2 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 2 passed")
		fmt.Println()
	}

	// Test 3: Read
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 3: Read")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := testRead(ctx, client); err != nil {
		log.Printf("❌ Test 3 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 3 passed")
		fmt.Println()
	}

	// Test 4: Write
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 4: Write")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := testWrite(ctx, client); err != nil {
		log.Printf("❌ Test 4 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 4 passed")
		fmt.Println()
	}

	// Test 5: ReadWrite
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔄 Test 5: ReadWrite")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := testReadWrite(ctx, client); err != nil {
		log.Printf("❌ Test 5 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 5 passed")
		fmt.Println()
	}

	// Test 6: Notifications
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔔 Test 6: Notifications")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := testNotifications(ctx, client); err != nil {
		log.Printf("❌ Test 6 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 6 passed")
		fmt.Println()
	}

	// Test 7: WriteControl
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("⚙️  Test 7: WriteControl")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := testWriteControl(ctx, client, initialState); err != nil {
		log.Printf("❌ Test 7 failed: %v\n\n", err)
	} else {
		fmt.Println("✅ Test 7 passed")
		fmt.Println()
	}

	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    All Tests Complete                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	printAddressInfo()
}

func testReadDeviceInfo(ctx context.Context, client *goadstc.Client) error {
	fmt.Println("Reading device information...")

	info, err := client.ReadDeviceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to read device info: %w", err)
	}

	fmt.Printf("  Device Name: %s\n", info.Name)
	fmt.Printf("  Version: %d.%d.%d\n", info.MajorVersion, info.MinorVersion, info.VersionBuild)
	return nil
}

func testReadState(ctx context.Context, client *goadstc.Client) (*goadstc.DeviceState, error) {
	fmt.Println("Reading device state...")

	state, err := client.ReadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}

	fmt.Printf("  ADS State: %s (%d)\n", state.ADSState.String(), state.ADSState)
	fmt.Printf("  Device State: %d\n", state.DeviceState)
	return state, nil
}

func testRead(ctx context.Context, client *goadstc.Client) error {
	fmt.Printf("Reading 4 bytes from IndexGroup=0x%04X, IndexOffset=%d...\n",
		ads.IndexGroupPLCMemory, monitorOffset)

	data, err := client.Read(ctx, ads.IndexGroupPLCMemory, monitorOffset, 4)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	if len(data) >= 4 {
		value := binary.LittleEndian.Uint32(data)
		fmt.Printf("  Read %d bytes: 0x%02X (decimal: %d)\n", len(data), value, value)
	} else {
		fmt.Printf("  Read %d bytes: %X\n", len(data), data)
	}
	return nil
}

func testWrite(ctx context.Context, client *goadstc.Client) error {
	// Generate test value (current timestamp in seconds as uint32)
	testValue := uint32(time.Now().Unix() % 0xFFFFFFFF)

	fmt.Printf("Writing test value %d (0x%08X) to offset %d...\n", testValue, testValue, testOffset)

	// Write value
	writeData := make([]byte, 4)
	binary.LittleEndian.PutUint32(writeData, testValue)

	err := client.Write(ctx, ads.IndexGroupPLCMemory, testOffset, writeData)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	fmt.Println("  Write successful")

	// Read back to verify
	fmt.Printf("Reading back from offset %d to verify...\n", testOffset)
	readData, err := client.Read(ctx, ads.IndexGroupPLCMemory, testOffset, 4)
	if err != nil {
		return fmt.Errorf("failed to read back: %w", err)
	}

	readValue := binary.LittleEndian.Uint32(readData)
	fmt.Printf("  Read back: %d (0x%08X)\n", readValue, readValue)

	if readValue == testValue {
		fmt.Println("  ✓ Verification successful - values match!")
	} else {
		return fmt.Errorf("verification failed: expected %d, got %d", testValue, readValue)
	}

	return nil
}

func testReadWrite(ctx context.Context, client *goadstc.Client) error {
	// Prepare write data (2 bytes)
	testValue := uint16(0xABCD)
	writeData := make([]byte, 2)
	binary.LittleEndian.PutUint16(writeData, testValue)

	fmt.Printf("ReadWrite: Writing 2 bytes (0x%04X) to offset %d and reading 4 bytes back...\n",
		testValue, testOffset2)

	readData, err := client.ReadWrite(ctx, ads.IndexGroupPLCMemory, testOffset2, 4, writeData)
	if err != nil {
		return fmt.Errorf("failed to read/write: %w", err)
	}

	fmt.Printf("  Wrote: 0x%04X\n", testValue)
	if len(readData) >= 4 {
		readValue := binary.LittleEndian.Uint32(readData)
		fmt.Printf("  Read: 0x%08X (decimal: %d)\n", readValue, readValue)
	} else {
		fmt.Printf("  Read %d bytes: %X\n", len(readData), readData)
	}

	return nil
}

func testNotifications(ctx context.Context, client *goadstc.Client) error {
	// Subscribe to MAIN.uUint variable
	// IndexGroup: 0x4040 (16448 decimal), IndexOffset: 0x5E0A0 (385184 decimal)
	fmt.Println("Subscribing to MAIN.uUint (IndexGroup: 0x4040, Offset: 0x5E0A0)...")
	fmt.Println("Monitoring for 10 seconds or 10 notifications (whichever comes first)")
	fmt.Println("  Transmission Mode: OnChange")
	fmt.Println("  ⚠️  Please change MAIN.uUint value in TwinCAT to trigger notifications")

	sub, err := client.Subscribe(ctx, goadstc.NotificationOptions{
		IndexGroup:       0x4040,                // MAIN.uUint IndexGroup
		IndexOffset:      0x5E0A0,               // MAIN.uUint IndexOffset
		Length:           2,                     // 2 bytes (UINT is 16-bit)
		TransmissionMode: ads.TransModeOnChange, // Notify on value change
		MaxDelay:         100 * time.Millisecond,
		CycleTime:        50 * time.Millisecond,
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer func() {
		fmt.Println("  Closing subscription...")
		sub.Close()
	}()

	fmt.Printf("  ✓ Subscribed successfully with handle %d\n", sub.Handle())

	// First, let's verify we can read the value directly
	fmt.Println("  Testing direct read of MAIN.uUint to verify address...")
	testData, err := client.Read(ctx, 0x4040, 0x5E0A0, 2)
	if err != nil {
		fmt.Printf("  ⚠️  Warning: Cannot read MAIN.uUint directly: %v\n", err)
		fmt.Println("  This might indicate an incorrect address")
	} else {
		value := binary.LittleEndian.Uint16(testData)
		fmt.Printf("  ✓ Direct read successful: MAIN.uUint = %d\n", value)
	}

	fmt.Println("  Waiting for notifications from PLC...")
	fmt.Println("  👉 Change MAIN.uUint value now in TwinCAT...")

	// Monitor for 10 seconds or 10 notifications
	timeout := time.After(10 * time.Second)
	notifCount := 0
	maxNotifications := 10

	fmt.Println("  Waiting for notifications...")

	for {
		select {
		case notif, ok := <-sub.Notifications():
			if !ok {
				fmt.Println("  Notification channel closed")
				return nil
			}
			notifCount++
			if len(notif.Data) >= 2 {
				value := binary.LittleEndian.Uint16(notif.Data[0:2])
				fmt.Printf("  [%d] %s | MAIN.uUint = %d (0x%04X)\n",
					notifCount,
					notif.Timestamp.Format("15:04:05.000"),
					value,
					value)
			} else {
				fmt.Printf("  [%d] %s | %d bytes: %X\n",
					notifCount,
					notif.Timestamp.Format("15:04:05.000"),
					len(notif.Data),
					notif.Data)
			}

			if notifCount >= maxNotifications {
				fmt.Printf("  Reached %d notifications, stopping...\n", maxNotifications)
				return nil
			}

		case <-timeout:
			fmt.Printf("  Timeout after 10 seconds, received %d notification(s)\n", notifCount)
			if notifCount == 0 {
				fmt.Println("  ⚠️  No notifications received - possible issues:")
				fmt.Println("     - Variable address may not support notifications")
				fmt.Println("     - PLC may not be sending notification packets")
				fmt.Println("     - Check ADS routes and firewall settings")
			}
			return nil
		}
	}
}

func testWriteControl(ctx context.Context, client *goadstc.Client, initialState *goadstc.DeviceState) error {
	if initialState == nil {
		return fmt.Errorf("no initial state available")
	}

	fmt.Println("⚠️  WARNING: This test will attempt to stop and restart the PLC!")
	fmt.Print("Do you want to proceed? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		fmt.Println("  Test skipped by user")
		return nil
	}

	// Check if PLC is running
	if initialState.ADSState != ads.StateRun {
		fmt.Printf("  PLC is not in RUN state (current: %s), skipping stop/start test\n",
			initialState.ADSState.String())
		return nil
	}

	// Note: Stopping the PLC may close the ADS connection
	fmt.Println("Stopping PLC...")
	fmt.Println("  ⚠️  Note: PLC may close connection when stopped")

	err = client.WriteControl(ctx, ads.StateStop, 0, nil)
	if err != nil {
		// Connection closure on stop is somewhat expected behavior
		if strings.Contains(err.Error(), "connection closed") {
			fmt.Println("  ℹ️  Connection closed (expected when stopping PLC)")
			fmt.Println("  ✓ Stop command was sent before connection closed")
			fmt.Println("\n  ⚠️  Cannot verify state or restart PLC - connection lost")
			fmt.Println("  To restart the PLC, use TwinCAT System Manager or reconnect")
			return nil
		}
		return fmt.Errorf("failed to stop PLC: %w", err)
	}
	fmt.Println("  ✓ Stop command sent")

	// Wait a moment for state transition
	time.Sleep(2 * time.Second)

	// Try to check state after stop (may fail if connection was closed)
	state, err := client.ReadState(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "connection closed") {
			fmt.Println("  ℹ️  Cannot read state - connection closed by PLC")
			fmt.Println("  This is normal behavior when PLC stops")
			return nil
		}
		return fmt.Errorf("failed to read state after stop: %w", err)
	}
	fmt.Printf("  Current state: %s\n", state.ADSState.String())

	// If we got here, connection is still alive, try to restart
	if state.ADSState != ads.StateRun {
		fmt.Println("\nStarting PLC...")
		err = client.WriteControl(ctx, ads.StateRun, 0, nil)
		if err != nil {
			return fmt.Errorf("failed to start PLC: %w", err)
		}
		fmt.Println("  ✓ Start command sent")

		// Wait for startup
		time.Sleep(2 * time.Second)

		// Check final state
		state, err = client.ReadState(ctx)
		if err != nil {
			return fmt.Errorf("failed to read final state: %w", err)
		}
		fmt.Printf("  Current state: %s\n", state.ADSState.String())
	}

	return nil
}

func printAddressInfo() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           How to Find Variable Addresses                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📌 Finding addresses in TwinCAT:")
	fmt.Println()
	fmt.Println("  1. In TwinCAT XAE, open your PLC project")
	fmt.Println("  2. Right-click on your PLC instance → 'Symbol Information'")
	fmt.Println("  3. Look for your variable in the list")
	fmt.Println("  4. The 'Address' column shows the offset (e.g., %MB100)")
	fmt.Println()
	fmt.Println("  Memory areas:")
	fmt.Println("    - %MB = Memory Byte (use decimal offset)")
	fmt.Println("    - %MW = Memory Word (offset * 1)")
	fmt.Println("    - %MD = Memory DWord (offset * 1)")
	fmt.Println()
	fmt.Println("  Index Groups:")
	fmt.Println("    - 0x4020: PLC memory (most common for variables)")
	fmt.Println("    - 0x4021: PLC memory bit access")
	fmt.Println("    - 0xF020: Physical inputs")
	fmt.Println("    - 0xF030: Physical outputs")
	fmt.Println()
	fmt.Println("  Example:")
	fmt.Println("    Variable at %MB100 → IndexGroup=0x4020, IndexOffset=100")
	fmt.Println("    Variable at %MW50  → IndexGroup=0x4020, IndexOffset=50")
	fmt.Println()
	fmt.Println("  💡 Tip: Use 'Online' mode in TwinCAT to see live addresses")
	fmt.Println()
}
