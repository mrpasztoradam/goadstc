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
	printHeader()
	client, ctx := setupClient()
	defer client.Close()

	runTest1ReadIntArray(ctx, client)
	runTest2WriteToArray(ctx, client)
	runTest3ReadStructArray(ctx, client)
	runTest4WriteStructArray(ctx, client)
	runTest5StringOperations(ctx, client)

	printSummary()
}

func printHeader() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║       Array Element Access Example (Milestone 5)        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func setupClient() (*goadstc.Client, context.Context) {
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
	fmt.Println("✅ Connected successfully")
	return client, context.Background()
}

func runTest1ReadIntArray(ctx context.Context, client *goadstc.Client) {
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
	fmt.Println("✅ Test 1 complete")
	fmt.Println()
}

func runTest2WriteToArray(ctx context.Context, client *goadstc.Client) {
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
	fmt.Println("✅ Test 2 complete")
	fmt.Println()
}

func runTest3ReadStructArray(ctx context.Context, client *goadstc.Client) {
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
	fmt.Println("✅ Test 3 complete")
	fmt.Println()
}

func runTest4WriteStructArray(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 4: Write to Struct Array Elements")
	fmt.Println("═══════════════════════════════════════════════════════════")

	writeStructFields0(ctx, client)
	writeStructFields1(ctx, client)
	verifyStructWrites(ctx, client)

	fmt.Println("✅ Test 4 complete")
	fmt.Println()
}

func writeStructFields0(ctx context.Context, client *goadstc.Client) {
	fmt.Println("   Writing to MAIN.aStruct[0] fields using type-safe methods...")
	if err := client.WriteUint16(ctx, "MAIN.aStruct[0].uiTest", 999); err != nil {
		log.Printf("   ⚠️  Failed to write uiTest: %v", err)
	} else {
		fmt.Println("   ✅ Wrote 999 to MAIN.aStruct[0].uiTest")
	}

	if err := client.WriteInt16(ctx, "MAIN.aStruct[0].iTest", -123); err != nil {
		log.Printf("   ⚠️  Failed to write iTest: %v", err)
	} else {
		fmt.Println("   ✅ Wrote -123 to MAIN.aStruct[0].iTest")
	}
}

func writeStructFields1(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n   Writing to MAIN.aStruct[1] fields...")
	if err := client.WriteUint16(ctx, "MAIN.aStruct[1].uiTest", 777); err != nil {
		log.Printf("   ⚠️  Failed to write uiTest: %v", err)
	} else {
		fmt.Println("   ✅ Wrote 777 to MAIN.aStruct[1].uiTest")
	}

	if err := client.WriteInt16(ctx, "MAIN.aStruct[1].iTest", 456); err != nil {
		log.Printf("   ⚠️  Failed to write iTest: %v", err)
	} else {
		fmt.Println("   ✅ Wrote 456 to MAIN.aStruct[1].iTest")
	}
}

func verifyStructWrites(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n   Verifying writes by reading both structs...")
	for i := 0; i < 2; i++ {
		symbolName := fmt.Sprintf("MAIN.aStruct[%d]", i)
		structData, err := client.ReadStructAsMap(ctx, symbolName)
		if err != nil {
			log.Printf("   ⚠️  Failed to read %s: %v", symbolName, err)
			continue
		}

		jsonData, _ := json.MarshalIndent(structData, "   ", "  ")
		fmt.Printf("   %s after write:\n%s\n", symbolName, string(jsonData))
	}
}

func runTest5StringOperations(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📝 Test 5: String Operations")
	fmt.Println("═══════════════════════════════════════════════════════════")

	testBasicStringOps(ctx, client)
	testStructArrayString(ctx, client)

	fmt.Println("✅ Test 5 complete")
	fmt.Println()
}

func testBasicStringOps(ctx context.Context, client *goadstc.Client) {
	fmt.Println("   Reading MAIN.sString...")
	strValue, err := client.ReadString(ctx, "MAIN.sString")
	if err != nil {
		log.Printf("   ⚠️  Failed to read: %v", err)
	} else {
		fmt.Printf("   Current value: %q\n", strValue)
	}

	fmt.Println("\n   Writing \"Hello Arrays!\" to MAIN.sString...")
	if err := client.WriteString(ctx, "MAIN.sString", "Hello Arrays!"); err != nil {
		log.Printf("   ⚠️  Failed to write: %v", err)
	} else {
		fmt.Println("   ✅ Write successful")
	}

	fmt.Println("\n   Verifying write...")
	strValue, err = client.ReadString(ctx, "MAIN.sString")
	if err != nil {
		log.Printf("   ⚠️  Failed to read: %v", err)
	} else {
		fmt.Printf("   New value: %q", strValue)
		if strValue == "Hello Arrays!" {
			fmt.Println(" ✅")
		} else {
			fmt.Println(" ❌")
		}
	}
}

func testStructArrayString(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n   Writing to string field in struct array...")
	if err := client.WriteString(ctx, "MAIN.aStruct[0].sTest", "Array String!"); err != nil {
		log.Printf("   ⚠️  Failed to write: %v", err)
	} else {
		fmt.Println("   ✅ Wrote \"Array String!\" to MAIN.aStruct[0].sTest")
	}

	fmt.Println("\n   Reading back struct with string field...")
	structData, err := client.ReadStructAsMap(ctx, "MAIN.aStruct[0]")
	if err != nil {
		log.Printf("   ⚠️  Failed to read: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(structData, "   ", "  ")
		fmt.Printf("   MAIN.aStruct[0]:\n%s\n", string(jsonData))
	}
}

func printSummary() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 5 Complete!                       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("✨ Array Element Access Features:")
	fmt.Println()
	fmt.Println("  ✅ Read array elements using bracket notation")
	fmt.Println("  ✅ Write to specific array indices")
	fmt.Println("  ✅ Works with primitive types (INT, UINT, etc.)")
	fmt.Println("  ✅ Works with struct arrays (read and write)")
	fmt.Println("  ✅ Write to struct array fields using dot notation")
	fmt.Println("  ✅ String read/write operations")
	fmt.Println("  ✅ Strings in struct arrays")
	fmt.Println("  ✅ Automatic offset calculation")
	fmt.Println("  ✅ Type-safe operations")
	fmt.Println()
	fmt.Println("  Usage:")
	fmt.Println("    client.ReadInt16(ctx, \"MAIN.myArray[5]\")")
	fmt.Println("    client.WriteInt16(ctx, \"MAIN.myArray[5]\", value)")
	fmt.Println("    client.ReadString(ctx, \"MAIN.myString\")")
	fmt.Println("    client.WriteString(ctx, \"MAIN.myString\", \"value\")")
	fmt.Println("    client.ReadStructAsMap(ctx, \"MAIN.structArray[2]\")")
	fmt.Println("    client.WriteUint16(ctx, \"MAIN.structArray[2].field\", value)")
	fmt.Println("    client.WriteString(ctx, \"MAIN.structArray[2].text\", \"value\")")
}
