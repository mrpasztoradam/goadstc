package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║       Array Element Access Example (Milestone 5)        ║")
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

	// Test 1: Read array of INT elements
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 1: Read Array of INT Elements")
	fmt.Println("═══════════════════════════════════════════════════════════")
	
	fmt.Println("   Reading MAIN.aInt[0], MAIN.aInt[1], MAIN.aInt[2]...")
	for i := 0; i < 3; i++ {
		symbolName := fmt.Sprintf("MAIN.aInt[%d]", i)
		value, err := client.ReadInt16(ctx, symbolName)
		if err != nil {
			log.Printf("   ⚠️  Failed to read %s: %v", symbolName, err)
		} else {
			fmt.Printf("   %s = %d\n", symbolName, value)
		}
	}
	fmt.Println("✅ Test 1 complete\n")

	// Test 2: Write to array elements
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 2: Write to Array Elements")
	fmt.Println("═══════════════════════════════════════════════════════════")
	
	testValues := []int16{100, 200, 300}
	fmt.Println("   Writing values [100, 200, 300] to MAIN.aInt[0..2]...")
	for i, val := range testValues {
		symbolName := fmt.Sprintf("MAIN.aInt[%d]", i)
		if err := client.WriteInt16(ctx, symbolName, val); err != nil {
			log.Printf("   ⚠️  Failed to write %s: %v", symbolName, err)
		} else {
			fmt.Printf("   ✅ Wrote %d to %s\n", val, symbolName)
		}
	}
	
	fmt.Println("\n   Verifying writes...")
	for i := 0; i < 3; i++ {
		symbolName := fmt.Sprintf("MAIN.aInt[%d]", i)
		value, err := client.ReadInt16(ctx, symbolName)
		if err != nil {
			log.Printf("   ⚠️  Failed to read %s: %v", symbolName, err)
		} else {
			fmt.Printf("   %s = %d", symbolName, value)
			if value == testValues[i] {
				fmt.Println(" ✅")
			} else {
				fmt.Printf(" ❌ (expected %d)\n", testValues[i])
			}
		}
	}
	fmt.Println("✅ Test 2 complete\n")

	// Test 3: Read array of struct elements
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 3: Read Array of Struct Elements")
	fmt.Println("═══════════════════════════════════════════════════════════")
	
	fmt.Println("   Reading MAIN.aStruct[0] and MAIN.aStruct[1]...")
	for i := 0; i < 2; i++ {
		symbolName := fmt.Sprintf("MAIN.aStruct[%d]", i)
		structData, err := client.ReadStructAsMap(ctx, symbolName)
		if err != nil {
			log.Printf("   ⚠️  Failed to read %s: %v", symbolName, err)
			continue
		}
		
		jsonData, _ := json.MarshalIndent(structData, "   ", "  ")
		fmt.Printf("   %s:\n%s\n\n", symbolName, string(jsonData))
	}
	fmt.Println("✅ Test 3 complete\n")

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 5 Complete!                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("✨ Array Element Access Features:")
	fmt.Println()
	fmt.Println("  ✅ Read array elements using bracket notation")
	fmt.Println("  ✅ Write to specific array indices")
	fmt.Println("  ✅ Works with primitive types (INT, UINT, etc.)")
	fmt.Println("  ✅ Works with struct arrays")
	fmt.Println("  ✅ Automatic offset calculation")
	fmt.Println("  ✅ Type-safe operations")
	fmt.Println()
	fmt.Println("  Usage:")
	fmt.Println("    client.ReadInt16(ctx, \"MAIN.myArray[5]\")")
	fmt.Println("    client.WriteInt16(ctx, \"MAIN.myArray[5]\", value)")
	fmt.Println("    client.ReadStructAsMap(ctx, \"MAIN.structArray[2]\")")
}
