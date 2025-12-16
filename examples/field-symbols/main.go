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
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Field Symbols Example                                 ║")
	fmt.Println("║    (Shows struct field access methods)                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Connect to PLC
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

	// First, check what symbols are actually available
	checkAvailableSymbols(ctx, client)

	// Demonstrate reading
	demonstrateReading(ctx, client)

	// Demonstrate writing
	demonstrateWriting(ctx, client)

	// Demonstrate subscriptions
	demonstrateSubscriptions(ctx, client)

	// Demonstrate type-agnostic operations
	demonstrateTypeAgnostic(ctx, client)
}

func checkAvailableSymbols(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Diagnostic: Checking Available Symbols")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// List all symbols to see what's actually registered
	fmt.Println("Fetching ALL symbols from PLC...")
	allSymbols, err := client.ListSymbols(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to list symbols: %v\n\n", err)
		return
	}

	// Filter for symbols containing "structExample2"
	fmt.Println("\nSymbols containing 'structExample2':")
	foundCount := 0
	for _, sym := range allSymbols {
		if len(sym.Name) > 0 && (sym.Name == "MAIN.structExample2" ||
			len(sym.Name) > len("MAIN.structExample2") && sym.Name[:len("MAIN.structExample2")] == "MAIN.structExample2") {
			fmt.Printf("  ✅ %s (size: %d bytes)\n", sym.Name, sym.Size)
			foundCount++
		}
	}

	if foundCount == 1 {
		fmt.Println("\n❌ ISSUE: Only the parent struct is registered!")
		fmt.Println("   Individual field symbols are NOT in the symbol table.")
		fmt.Println()
		fmt.Println("⚠️  TwinCAT does NOT have a {attribute 'symbol'} pragma.")
		fmt.Println("   By default, TwinCAT MAY expose struct fields as symbols depending on:")
		fmt.Println("   • Project settings (Symbol Configuration)")
		fmt.Println("   • Whether {attribute 'TcHideSubItems'} is used")
		fmt.Println("   • TwinCAT version behavior")
		fmt.Println()
		fmt.Println("💡 Solutions (in order of preference):")
		fmt.Println()
		fmt.Println("   1. ✅ Use WriteStructFields() - works without any PLC changes")
		fmt.Println("      err := client.WriteStructFields(ctx, \"MAIN.structExample2\", map[string]interface{}{")
		fmt.Println("          \"iTest\": int16(7777),")
		fmt.Println("      })")
		fmt.Println()
		fmt.Println("   2. Check Project Settings:")
		fmt.Println("      • TwinCAT Project → Properties → Symbol Configuration")
		fmt.Println("      • Enable \"Generate symbol info for sub-items\" or similar")
		fmt.Println()
		fmt.Println("   3. Create separate VAR_GLOBAL variables:")
		fmt.Println("      VAR_GLOBAL")
		fmt.Println("          structExample2_iTest : INT;  // Direct access")
		fmt.Println("      END_VAR")
		fmt.Println()
		fmt.Println("   4. Use REFERENCE variables (TwinCAT 3.1.4024+):")
		fmt.Println("      VAR_GLOBAL")
		fmt.Println("          structExample2 : ST_StructExample2;")
		fmt.Println("          refToITest : REFERENCE TO INT REF= structExample2.iTest;")
		fmt.Println("      END_VAR")
	} else if foundCount > 1 {
		fmt.Println("\n✅ Field symbols are registered! Continuing with examples...")
	} else {
		fmt.Println("\n⚠️  No symbols found matching 'structExample2'")
	}
	fmt.Println()
}

func demonstrateReading(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Step 1: Reading - Struct vs Fields")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("If field symbols are exposed, you can read either way:")
	fmt.Println()

	// Read entire struct
	fmt.Println("1️⃣  Reading entire struct:")
	structValue, err := client.ReadSymbolValue(ctx, "MAIN.structExample2")
	if err != nil {
		log.Fatalf("⚠️  Failed to read struct: %v", err)
	}

	if structMap, ok := structValue.(map[string]interface{}); ok {
		fmt.Println("   Result type: map[string]interface{}")
		for fieldName, fieldValue := range structMap {
			fmt.Printf("   - %s: %v\n", fieldName, fieldValue)
		}
	}
	fmt.Println()

	// Read individual fields directly
	fmt.Println("2️⃣  Reading individual fields directly:")

	iTestValue, err := client.ReadSymbolValue(ctx, "MAIN.structExample2.iTest")
	if err != nil {
		fmt.Printf("   ⚠️  MAIN.structExample2.iTest: not exposed as symbol\n")
		fmt.Println("   💡 Use WriteStructFields() method instead (see struct-manipulation example)")
	} else {
		fmt.Printf("   ✅ MAIN.structExample2.iTest = %v (type: %T)\n", iTestValue, iTestValue)
	}

	// Try nested field
	nestedValue, err := client.ReadSymbolValue(ctx, "MAIN.structExample2.stTest.iTest")
	if err != nil {
		fmt.Printf("   ⚠️  MAIN.structExample2.stTest.iTest: not exposed as symbol\n")
	} else {
		fmt.Printf("   ✅ MAIN.structExample2.stTest.iTest = %v (type: %T)\n", nestedValue, nestedValue)
	}
	fmt.Println()
}

func demonstrateWriting(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Step 2: Writing - Direct Field Access")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("With field symbols, write directly without struct manipulation:")
	fmt.Println()

	// Write to individual field
	fmt.Println("Writing MAIN.structExample2.iTest = 7777...")
	err := client.WriteSymbolValue(ctx, "MAIN.structExample2.iTest", int16(7777))
	if err != nil {
		fmt.Printf("⚠️  Write failed: field not exposed as symbol\n")
		fmt.Println("💡 Use WriteStructFields() method instead (see struct-manipulation example)")
		fmt.Println()
		return
	}
	fmt.Println("✅ Write successful!")
	fmt.Println()

	// Verify the write
	fmt.Println("Verifying write by reading back:")
	value, err := client.ReadSymbolValue(ctx, "MAIN.structExample2.iTest")
	if err != nil {
		fmt.Printf("⚠️  Read failed: %v\n", err)
	} else {
		fmt.Printf("✅ Confirmed: MAIN.structExample2.iTest = %v\n", value)
	}
	fmt.Println()
}

func demonstrateSubscriptions(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔔 Step 3: Subscriptions - Monitor Specific Fields")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("Subscribe to individual fields for efficient monitoring:")
	fmt.Println()

	// Subscribe to a specific field
	fmt.Println("Subscribing to MAIN.structExample2.iTest...")
	sub, err := client.SubscribeSymbol(ctx, "MAIN.structExample2.iTest", goadstc.SymbolNotificationOptions{
		TransmissionMode: ads.TransModeOnChange,
		MaxDelay:         100 * time.Millisecond,
		CycleTime:        50 * time.Millisecond,
	})

	if err != nil {
		fmt.Printf("⚠️  Subscribe failed: %v\n", err)
		fmt.Println("💡 Field not exposed as individual symbol in PLC")
		fmt.Println("   Alternative: Subscribe to whole struct and parse fields from updates")
		fmt.Println()
		return
	}
	defer sub.Close()

	fmt.Println("✅ Subscribed! Monitoring for 3 seconds...")
	fmt.Println("   (Try changing the value in TwinCAT)")
	fmt.Println()

	// Monitor notifications
	changeCount := 0
	done := make(chan bool)

	go func() {
		for notif := range sub.Notifications() {
			changeCount++
			// Auto-parse the value
			if val, err := client.ReadSymbolValue(ctx, "MAIN.structExample2.iTest"); err == nil {
				fmt.Printf("   🔔 Change detected! iTest = %v (timestamp: %s)\n",
					val, notif.Timestamp.Format("15:04:05.000"))
			}
		}
	}()

	// Wait for changes
	time.Sleep(3 * time.Second)
	sub.Close()
	close(done)

	fmt.Printf("✅ Monitoring complete. Detected %d changes.\n", changeCount)
	fmt.Println()
}

func demonstrateTypeAgnostic(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🎯 Step 4: Type-Agnostic Operations")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("Read any symbol without knowing if it's a struct or field:")
	fmt.Println()

	// List of symbols to try - mix of struct and fields
	symbolsToTry := []string{
		"MAIN.structExample2",        // whole struct
		"MAIN.structExample2.iTest",  // field
		"MAIN.structExample2.stTest", // nested struct
	}

	for _, symbolName := range symbolsToTry {
		fmt.Printf("Reading '%s'...\n", symbolName)
		value, err := client.ReadSymbolValue(ctx, symbolName)
		if err != nil {
			fmt.Printf("  ⚠️  Error: %v\n", err)
			continue
		}

		// Handle any type
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("  ✅ Type: Struct with %d fields\n", len(v))
			for fname, fval := range v {
				fmt.Printf("     - %s: %v\n", fname, fval)
			}
		case []interface{}:
			fmt.Printf("  ✅ Type: Array with %d elements\n", len(v))
		case bool:
			fmt.Printf("  ✅ Type: BOOL = %v\n", v)
		case int8, int16, int32, int64, uint8, uint16, uint32, uint64:
			fmt.Printf("  ✅ Type: Integer = %v\n", v)
		case float32, float64:
			fmt.Printf("  ✅ Type: Float = %v\n", v)
		case string:
			fmt.Printf("  ✅ Type: String = %q\n", v)
		case time.Time:
			fmt.Printf("  ✅ Type: DateTime = %v\n", v.Format(time.RFC3339))
		case time.Duration:
			fmt.Printf("  ✅ Type: Duration = %v\n", v)
		default:
			fmt.Printf("  ✅ Type: %T = %v\n", v, v)
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Example Complete!")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("📝 Key Takeaways:")
	fmt.Println("   • ReadSymbolValue() auto-detects types for ANY symbol")
	fmt.Println("   • WriteSymbolValue() works for directly accessible symbols")
	fmt.Println("   • WriteStructFields() works for struct fields (no PLC changes needed)")
	fmt.Println("   • SubscribeSymbol() monitors accessible symbols")
	fmt.Println("   • Type-agnostic: no need to know if symbol is struct/field/primitive")
	fmt.Println()
	fmt.Println("⚠️  If field access failed:")
	fmt.Println("   TwinCAT doesn't expose struct fields as symbols by default.")
	fmt.Println()
	fmt.Println("   ✅ RECOMMENDED: Use WriteStructFields() method")
	fmt.Println("   See examples/struct-manipulation for working examples!")
	fmt.Println()
}
