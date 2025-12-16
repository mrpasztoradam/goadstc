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
	fmt.Println("║    Struct Manipulation Example                          ║")
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

	// Step 1: Read the struct with automatic type detection
	readStructExample(ctx, client)

	// Step 2: Write to all struct fields
	writeStructFields(ctx, client)

	// Step 3: Verify the writes
	verifyStructWrites(ctx, client)
	// Bonus: Demonstrate partial field writing
	demonstratePartialWrite(ctx, client)}

func readStructExample(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Step 1: Reading MAIN.structExample2 with Auto-Detection")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Read the entire struct - automatically detects type and parses all fields
	value, err := client.ReadSymbolValue(ctx, "MAIN.structExample2")
	if err != nil {
		log.Fatalf("⚠️  Failed to read struct: %v", err)
	}

	// The value should be a map[string]interface{} with all struct fields
	if structVal, ok := value.(map[string]interface{}); ok {
		fmt.Printf("✅ Successfully read struct with %d fields:\n", len(structVal))
		for fieldName, fieldValue := range structVal {
			fmt.Printf("  %s: %v (type: %T)\n", fieldName, fieldValue, fieldValue)
		}
	} else {
		log.Fatalf("❌ Unexpected type: expected struct (map), got %T", value)
	}
	fmt.Println()
}

func writeStructFields(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Step 2: Writing Struct Fields (Whole Struct Method)")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// First, read the struct to see current values
	value, err := client.ReadSymbolValue(ctx, "MAIN.structExample2")
	if err != nil {
		log.Fatalf("⚠️  Failed to read struct: %v", err)
	}

	structVal, ok := value.(map[string]interface{})
	if !ok {
		log.Fatalf("❌ Not a struct type")
	}

	fmt.Println("Current struct values:")
	for fieldName, fieldValue := range structVal {
		fmt.Printf("  %s: %v\n", fieldName, fieldValue)
	}

	fmt.Println("\nWriting multiple fields using WriteStructFields()...")
	fmt.Println("This method reads the entire struct, modifies fields at their")
	fmt.Println("byte offsets, and writes the whole struct back.")
	fmt.Println()

	// Prepare field values to write
	fieldsToWrite := make(map[string]interface{})
	
	for fieldName, fieldValue := range structVal {
		// Determine appropriate write value based on field type
		switch v := fieldValue.(type) {
		case bool:
			fieldsToWrite[fieldName] = !v // Toggle boolean
			fmt.Printf("  %s (BOOL): will write %v\n", fieldName, !v)

		case int8:
			fieldsToWrite[fieldName] = int8(42)
			fmt.Printf("  %s (INT8): will write %v\n", fieldName, int8(42))

		case uint8:
			fieldsToWrite[fieldName] = uint8(42)
			fmt.Printf("  %s (UINT8): will write %v\n", fieldName, uint8(42))

		case int16:
			fieldsToWrite[fieldName] = int16(1234)
			fmt.Printf("  %s (INT16): will write %v\n", fieldName, int16(1234))

		case uint16:
			fieldsToWrite[fieldName] = uint16(5678)
			fmt.Printf("  %s (UINT16): will write %v\n", fieldName, uint16(5678))

		case int32:
			fieldsToWrite[fieldName] = int32(123456)
			fmt.Printf("  %s (INT32): will write %v\n", fieldName, int32(123456))

		case uint32:
			fieldsToWrite[fieldName] = uint32(987654)
			fmt.Printf("  %s (UINT32): will write %v\n", fieldName, uint32(987654))

		case int64:
			fieldsToWrite[fieldName] = int64(123456789)
			fmt.Printf("  %s (INT64): will write %v\n", fieldName, int64(123456789))

		case uint64:
			fieldsToWrite[fieldName] = uint64(987654321)
			fmt.Printf("  %s (UINT64): will write %v\n", fieldName, uint64(987654321))

		case float32:
			fieldsToWrite[fieldName] = float32(3.14159)
			fmt.Printf("  %s (REAL): will write %v\n", fieldName, float32(3.14159))

		case float64:
			fieldsToWrite[fieldName] = float64(2.71828)
			fmt.Printf("  %s (LREAL): will write %v\n", fieldName, float64(2.71828))

		case string:
			newVal := fmt.Sprintf("Updated_%s", fieldName)
			fieldsToWrite[fieldName] = newVal
			fmt.Printf("  %s (STRING): will write %q\n", fieldName, newVal)

		case time.Duration:
			fieldsToWrite[fieldName] = 500 * time.Millisecond
			fmt.Printf("  %s (TIME): will write %v\n", fieldName, 500*time.Millisecond)

		case time.Time:
			now := time.Now()
			fieldsToWrite[fieldName] = now
			fmt.Printf("  %s (DATE_AND_TIME): will write %v\n", fieldName, now.Format(time.RFC3339))

		case map[string]interface{}:
			fmt.Printf("  %s (nested struct): skipped (not yet supported)\n", fieldName)
			continue

		case []interface{}:
			fmt.Printf("  %s (array): skipped (not yet supported)\n", fieldName)
			continue

		default:
			fmt.Printf("  %s (unknown type %T): skipped\n", fieldName, v)
			continue
		}
	}

	// Write all fields at once
	if len(fieldsToWrite) > 0 {
		fmt.Printf("\n📝 Writing %d fields to MAIN.structExample2...\n", len(fieldsToWrite))
		if err := client.WriteStructFields(ctx, "MAIN.structExample2", fieldsToWrite); err != nil {
			log.Fatalf("⚠️  Failed to write struct fields: %v", err)
		}
		fmt.Println("✅ Successfully wrote all fields!")
	} else {
		fmt.Println("\n⚠️  No writable fields found")
	}
	fmt.Println()
}

func verifyStructWrites(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Step 3: Verifying Writes by Re-reading Struct")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Read the struct again to verify changes
	value, err := client.ReadSymbolValue(ctx, "MAIN.structExample2")
	if err != nil {
		log.Fatalf("⚠️  Failed to read: %v", err)
	}

	if structVal, ok := value.(map[string]interface{}); ok {
		fmt.Println("Updated struct values:")
		for fieldName, fieldValue := range structVal {
			// Format output based on type
			switch v := fieldValue.(type) {
			case time.Time:
				fmt.Printf("  %s: %v ✅\n", fieldName, v.Format(time.RFC3339))
			case time.Duration:
				fmt.Printf("  %s: %v ✅\n", fieldName, v)
			case map[string]interface{}:
				fmt.Printf("  %s: <nested struct with %d fields>\n", fieldName, len(v))
			default:
				fmt.Printf("  %s: %v ✅\n", fieldName, fieldValue)
			}
		}
		fmt.Println("\n✅ Example completed successfully!")
		fmt.Println("\n📝 This example demonstrated:")
		fmt.Println("   ✓ Reading structs with automatic type detection")
		fmt.Println("   ✓ Discovering field types dynamically")
		fmt.Println("   ✓ Writing multiple struct fields in a single operation")
		fmt.Println("   ✓ Using byte-offset modification for field writes")
	}
}

// demonstratePartialWrite shows writing only specific fields
func demonstratePartialWrite(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🎯 Bonus: Writing Only Specific Fields")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Write just one or two fields
	fmt.Println("Writing only the 'iTest' field to 9999...")
	err := client.WriteStructFields(ctx, "MAIN.structExample2", map[string]interface{}{
		"iTest": int16(9999),
	})
	if err != nil {
		log.Fatalf("⚠️  Failed to write: %v", err)
	}
	fmt.Println("✅ Successfully wrote partial update!")

	// Verify
	value, err := client.ReadSymbolValue(ctx, "MAIN.structExample2")
	if err != nil {
		log.Fatalf("⚠️  Failed to read: %v", err)
	}

	if structVal, ok := value.(map[string]interface{}); ok {
		fmt.Println("\nFinal struct values:")
		for fieldName, fieldValue := range structVal {
			fmt.Printf("  %s: %v\n", fieldName, fieldValue)
		}
	}
	fmt.Println()
}
