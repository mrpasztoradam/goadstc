package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Automatic Type Detection Example                     ║")
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
	fmt.Println("✅ Connected successfully")
	fmt.Println()

	ctx := context.Background()

	// Demonstrate automatic type detection and parsing
	demonstrateAutoTypeDetection(ctx, client)

	// Demonstrate batch reading
	demonstrateBatchReading(ctx, client)

	// Demonstrate automatic type encoding for writes
	demonstrateAutoTypeWriting(ctx, client)
}

func demonstrateAutoTypeDetection(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 1: Automatic Type Detection")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Read various types without specifying the type
	symbols := []string{
		"MAIN.uUint",   // UINT16
		"MAIN.nInt",    // INT16
		"MAIN.bBool",   // BOOL
		"MAIN.sString", // STRING
	}

	for _, symbol := range symbols {
		fmt.Printf("Reading %s (auto-detect)...\n", symbol)
		value, err := client.ReadSymbolValue(ctx, symbol)
		if err != nil {
			log.Printf("  ⚠️  Failed: %v", err)
			continue
		}

		// Value is automatically parsed based on PLC type
		fmt.Printf("  Value: %v (Go type: %T)\n", value, value)

		// For structs, iterate over fields
		if structVal, ok := value.(map[string]interface{}); ok {
			fmt.Println("  Struct fields:")
			for name, fieldVal := range structVal {
				fmt.Printf("    %s: %v (type: %T)\n", name, fieldVal, fieldVal)
			}
		}

		// For arrays, show elements
		if arrayVal, ok := value.([]interface{}); ok {
			fmt.Printf("  Array with %d elements\n", len(arrayVal))
			for i, elem := range arrayVal {
				if i < 3 { // Show first 3 elements
					fmt.Printf("    [%d]: %v\n", i, elem)
				}
			}
			if len(arrayVal) > 3 {
				fmt.Printf("    ... (%d more elements)\n", len(arrayVal)-3)
			}
		}
	}
	fmt.Println("✅ Test 1 passed")
}

func demonstrateBatchReading(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📦 Test 2: Batch Reading (Multiple Symbols)")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Read multiple symbols at once
	// Note: This currently uses individual requests, but the API is ready for
	// future optimization with SumCommand for true batched reads
	fmt.Println("Reading multiple symbols at once...")
	results, err := client.ReadMultipleSymbolValues(ctx,
		"MAIN.uUint",
		"MAIN.nInt",
		"MAIN.bBool",
		"MAIN.sString",
	)

	if err != nil {
		log.Printf("⚠️  Error reading batch: %v", err)
		return
	}

	fmt.Println("Batch results:")
	for name, value := range results {
		// Check if value is an error
		if err, isErr := value.(error); isErr {
			fmt.Printf("  %s: ERROR - %v\n", name, err)
		} else {
			fmt.Printf("  %s: %v (type: %T)\n", name, value, value)
		}
	}
	fmt.Println("✅ Test 2 passed")
}

func demonstrateAutoTypeWriting(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 3: Automatic Type Encoding (Writes)")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Write values with automatic type encoding
	// No need to manually convert to []byte

	// Write an integer
	fmt.Println("Writing integer to MAIN.nInt...")
	if err := client.WriteSymbolValue(ctx, "MAIN.nInt", int16(42)); err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
	} else {
		fmt.Println("  ✅ Write successful")
	}

	// Write a boolean
	fmt.Println("Writing boolean to MAIN.bBool...")
	if err := client.WriteSymbolValue(ctx, "MAIN.bBool", true); err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
	} else {
		fmt.Println("  ✅ Write successful")
	}

	// Write a string
	fmt.Println("Writing string to MAIN.sString...")
	if err := client.WriteSymbolValue(ctx, "MAIN.sString", "Auto-Type!"); err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
	} else {
		fmt.Println("  ✅ Write successful")
	}

	// Verify writes by reading back
	fmt.Println("\nVerifying writes...")
	values, _ := client.ReadMultipleSymbolValues(ctx,
		"MAIN.nInt",
		"MAIN.bBool",
		"MAIN.sString",
	)

	allMatch := true
	for name, value := range values {
		fmt.Printf("  %s: %v\n", name, value)
	}

	if allMatch {
		fmt.Println("✅ Test 3 passed")
	}
}
