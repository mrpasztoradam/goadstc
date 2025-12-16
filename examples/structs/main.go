package main

import (
	"context"
	"encoding/binary"
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

	runTest1ReadEntireStruct(ctx, client)
	runTest2FindStructSymbols(ctx, client)
	runTest3ReadStructAsMap(ctx, client)
	runTest4ManualFieldParsing(ctx, client)
	runTest5ParseNestedStruct(ctx, client)
	runTest6GetStructSymbolInfo(ctx, client)
	runTest7AutomaticParsing(ctx, client)

	printSummary()
}

func printHeader() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Struct Field Access Example (Milestone 4)            ║")
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

func runTest1ReadEntireStruct(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 1: Read Entire Struct")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structData, err := client.ReadSymbol(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  Read %d bytes from MAIN.structExample\n", len(structData))
	fmt.Printf("  Data (hex): %x...\n", structData[:min(32, len(structData))])
	fmt.Println("✅ Test 1 passed")
}

func runTest2FindStructSymbols(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 2: Find Struct Symbols")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structs, err := client.FindSymbols(ctx, "struct")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  Found %d struct symbols:\n", len(structs))
	for _, s := range structs {
		fmt.Printf("    - %s (%s, %d bytes)\n", s.Name, s.Type.Name, s.Size)
		if s.Type.IsStruct {
			fmt.Printf("      └─ Struct type detected\n")
		}
	}
	fmt.Println("✅ Test 2 passed")
}

func runTest3ReadStructAsMap(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📦 Test 3: Read Struct as Map (Parse Fields)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structMap, err := client.ReadStructAsMap(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  Parsed %d fields from MAIN.structExample:\n", len(structMap))
	for key, value := range structMap {
		fmt.Printf("    %s: %v\n", key, value)
	}
	fmt.Println("✅ Test 3 passed")
}

func runTest4ManualFieldParsing(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔧 Test 4: Manual Field Parsing (Known Offsets)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  For MAIN.structExample (TestSt), assuming structure:")
	fmt.Println("    Offset 0:  UINT (2 bytes)")
	fmt.Println("    Offset 2:  UINT (2 bytes)")
	fmt.Println("    Offset 4:  STRING (remaining bytes)")
	fmt.Println()

	data, err := client.ReadSymbol(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
		return
	}

	fmt.Println("  Manually parsed fields:")
	parseField1(data)
	parseField2(data)
	parseField3(data)
	fmt.Println("✅ Test 4 passed")
}

func parseField1(data []byte) {
	if len(data) >= 2 {
		field1 := binary.LittleEndian.Uint16(data[0:2])
		fmt.Printf("    Field1 (UINT @ offset 0): %d (0x%04X)\n", field1, field1)
	}
}

func parseField2(data []byte) {
	if len(data) >= 4 {
		field2 := binary.LittleEndian.Uint16(data[2:4])
		fmt.Printf("    Field2 (UINT @ offset 2): %d (0x%04X)\n", field2, field2)
	}
}

func parseField3(data []byte) {
	if len(data) > 4 {
		stringData := data[4:]
		for i, b := range stringData {
			if b == 0 {
				stringData = stringData[:i]
				break
			}
		}
		fmt.Printf("    Field3 (STRING @ offset 4): %q\n", string(stringData))
	}
}

func runTest5ParseNestedStruct(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔧 Test 5: Parse Nested Struct (MAIN.structExample2)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Reading MAIN.structExample2 (nestedSt, 88 bytes)...")

	nestedData, err := client.ReadSymbol(ctx, "MAIN.structExample2")
	if err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
		return
	}

	fmt.Printf("  Read %d bytes\n", len(nestedData))
	fmt.Printf("  Raw data (first 32 bytes): %x...\n", nestedData[:min(32, len(nestedData))])

	parseNestedStructWithMap(ctx, client)
	parseNestedStructManually(nestedData)
	fmt.Println("✅ Test 5 passed")
}

func parseNestedStructWithMap(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n  Parsing with ReadStructAsMap:")
	nestedMap, err := client.ReadStructAsMap(ctx, "MAIN.structExample2")
	if err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  Parsed fields:\n")
	for key, value := range nestedMap {
		if key == "_raw" {
			continue
		}
		fmt.Printf("    %s: %v\n", key, value)
	}
}

func parseNestedStructManually(nestedData []byte) {
	fmt.Println("\n  Exploring structure (nestedSt = INT + embedded TestSt):")
	offset := 0

	// Parse outer INT16 field
	if len(nestedData) >= 2 {
		val := int16(binary.LittleEndian.Uint16(nestedData[offset : offset+2]))
		fmt.Printf("    [%d] Outer.Field1 (INT): %d (0x%04X)\n", offset, val, uint16(val))
		offset += 2
	}

	// Parse embedded struct
	if offset < len(nestedData) {
		parseEmbeddedStruct(nestedData, offset)
	}
}

func parseEmbeddedStruct(nestedData []byte, offset int) {
	remaining := nestedData[offset:]
	fmt.Printf("\n    Embedded TestSt struct at offset %d (%d bytes remaining):\n", offset, len(remaining))

	// Parse embedded struct's first field (INT)
	if len(remaining) >= 2 {
		val1 := int16(binary.LittleEndian.Uint16(remaining[0:2]))
		fmt.Printf("      [%d] Inner.Field1 (INT): %d (0x%04X)\n", offset, val1, uint16(val1))
	}

	// Parse embedded struct's second field (UINT)
	if len(remaining) >= 4 {
		val2 := binary.LittleEndian.Uint16(remaining[2:4])
		fmt.Printf("      [%d] Inner.Field2 (UINT): %d (0x%04X)\n", offset+2, val2, val2)
	}

	// Parse embedded struct's STRING field
	if len(remaining) > 4 {
		parseEmbeddedString(remaining, offset)
	}
}

func parseEmbeddedString(remaining []byte, offset int) {
	nestedStringStart := 4
	nestedStringData := remaining[nestedStringStart:]
	for i, b := range nestedStringData {
		if b == 0 && i > 0 {
			fmt.Printf("      [%d] Inner.Field3 (STRING): %q\n",
				offset+nestedStringStart, string(nestedStringData[:i]))
			break
		}
	}
}

func runTest6GetStructSymbolInfo(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("ℹ️  Test 6: Get Struct Symbol Information")
	fmt.Println("═══════════════════════════════════════════════════════════")

	displayStructSymbol(client)
	displayNestedStructSymbol(client)
}

func displayStructSymbol(client *goadstc.Client) {
	structSym, err := client.GetSymbol("MAIN.structExample")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  Symbol: %s\n", structSym.Name)
	fmt.Printf("  Type: %s\n", structSym.Type.Name)
	fmt.Printf("  Size: %d bytes\n", structSym.Size)
	fmt.Printf("  IndexGroup: 0x%X\n", structSym.IndexGroup)
	fmt.Printf("  IndexOffset: 0x%X\n", structSym.IndexOffset)
	fmt.Printf("  IsStruct: %v\n", structSym.Type.IsStruct)
	fmt.Printf("  Fields defined: %d\n", len(structSym.Type.Fields))
	fmt.Println("✅ Test 6 passed")
}

func displayNestedStructSymbol(client *goadstc.Client) {
	nestedSym, err := client.GetSymbol("MAIN.structExample2")
	if err != nil {
		log.Printf("⚠️  Failed to get nested struct info: %v", err)
		return
	}
	fmt.Printf("\n  Nested struct info:\n")
	fmt.Printf("    Symbol: %s\n", nestedSym.Name)
	fmt.Printf("    Type: %s\n", nestedSym.Type.Name)
	fmt.Printf("    Size: %d bytes\n", nestedSym.Size)
	fmt.Printf("    IsStruct: %v\n", nestedSym.Type.IsStruct)
	fmt.Printf("    Fields defined: %d\n", len(nestedSym.Type.Fields))
}

func runTest7AutomaticParsing(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🤖 Test 7: Automatic Parsing from PLC")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  ReadStructAsMap will automatically fetch type info from PLC!")

	parseStructAutomatically(ctx, client, "MAIN.structExample", "TestSt")
	parseStructAutomatically(ctx, client, "MAIN.structExample2", "nestedSt")
	fmt.Println("\n  ✅ Test 7 passed")
}

func parseStructAutomatically(ctx context.Context, client *goadstc.Client, symbolName, structType string) {
	fmt.Printf("\n  Parsing %s (%s) automatically:\n", symbolName, structType)
	result, err := client.ReadStructAsMap(ctx, symbolName)
	if err != nil {
		log.Printf("  ⚠️  Failed to parse: %v", err)
		return
	}
	if structType == "TestSt" {
		fmt.Println("  ✅ Automatically discovered struct layout from PLC!")
	} else {
		fmt.Println("  ✅ Automatically discovered nested struct layout!")
	}
	printStructMap(result, "  ")
}

func printSummary() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 4 Summary                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("✨ Automatic Struct Parsing:")
	fmt.Println("  ReadStructAsMap automatically fetches type information from the PLC!")
	fmt.Println()
	fmt.Println("  How it works:")
	fmt.Println("  1. Call ReadStructAsMap(ctx, \"MAIN.myStruct\")")
	fmt.Println("  2. Library queries PLC for type definition (0xF011)")
	fmt.Println("  3. Type info is cached for subsequent calls")
	fmt.Println("  4. All fields are parsed automatically, including nested structs!")
	fmt.Println()
	fmt.Println("  No manual type registration needed!")
	fmt.Println("  No offset calculations required!")
	fmt.Println("  Works with nested structs automatically!")
	fmt.Println()
	fmt.Println("  Note: Requires TwinCAT 3 with data type upload support (0xF011).")
	fmt.Println("        If not supported, you can manually register types using")
	fmt.Println("        client.RegisterType(typeInfo) as a fallback.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func printStructMap(m map[string]interface{}, indent string) {
	for key, value := range m {
		if key == "_raw" {
			continue // Skip raw data
		}
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s (struct):\n", indent, key)
			printStructMap(v, indent+"  ")
		case string:
			fmt.Printf("%s%s: %q\n", indent, key, v)
		default:
			fmt.Printf("%s%s: %v\n", indent, key, v)
		}
	}
}
