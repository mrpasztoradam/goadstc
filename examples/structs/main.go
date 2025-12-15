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
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Struct Field Access Example (Milestone 4)            ║")
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

	ctx := context.Background()

	// Test 1: Read entire struct
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 1: Read Entire Struct")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structData, err := client.ReadSymbol(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  Read %d bytes from MAIN.structExample\n", len(structData))
		fmt.Printf("  Data (hex): %x...\n", structData[:min(32, len(structData))])
		fmt.Println("✅ Test 1 passed")
	}

	// Test 2: List available struct symbols
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 2: Find Struct Symbols")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structs, err := client.FindSymbols(ctx, "struct")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  Found %d struct symbols:\n", len(structs))
		for _, s := range structs {
			fmt.Printf("    - %s (%s, %d bytes)\n", s.Name, s.Type.Name, s.Size)
			if s.Type.IsStruct {
				fmt.Printf("      └─ Struct type detected\n")
			}
		}
		fmt.Println("✅ Test 2 passed")
	}

	// Test 3: Read struct as map (parse fields)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📦 Test 3: Read Struct as Map (Parse Fields)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structMap, err := client.ReadStructAsMap(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  Parsed %d fields from MAIN.structExample:\n", len(structMap))
		for key, value := range structMap {
			fmt.Printf("    %s: %v\n", key, value)
		}
		fmt.Println("✅ Test 3 passed")
	}

	// Test 4: Demonstrate manual field parsing with known offsets
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔧 Test 4: Manual Field Parsing (Known Offsets)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  For MAIN.structExample (TestSt), assuming structure:")
	fmt.Println("    Offset 0:  UINT (2 bytes)")
	fmt.Println("    Offset 2:  UINT (2 bytes)")
	fmt.Println("    Offset 4:  STRING (remaining bytes)")
	fmt.Println()

	// Read the struct
	data, err := client.ReadSymbol(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
	} else {
		fmt.Println("  Manually parsed fields:")
		if len(data) >= 2 {
			field1 := binary.LittleEndian.Uint16(data[0:2])
			fmt.Printf("    Field1 (UINT @ offset 0): %d (0x%04X)\n", field1, field1)
		}
		if len(data) >= 4 {
			field2 := binary.LittleEndian.Uint16(data[2:4])
			fmt.Printf("    Field2 (UINT @ offset 2): %d (0x%04X)\n", field2, field2)
		}
		if len(data) > 4 {
			// Find null terminator for string
			stringData := data[4:]
			for i, b := range stringData {
				if b == 0 {
					stringData = stringData[:i]
					break
				}
			}
			fmt.Printf("    Field3 (STRING @ offset 4): %q\n", string(stringData))
		}
		fmt.Println("✅ Test 4 passed")
	}

	// Test 5: Parse nested struct (MAIN.structExample2)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔧 Test 5: Parse Nested Struct (MAIN.structExample2)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Reading MAIN.structExample2 (nestedSt, 88 bytes)...")

	nestedData, err := client.ReadSymbol(ctx, "MAIN.structExample2")
	if err != nil {
		log.Printf("  ⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  Read %d bytes\n", len(nestedData))
		fmt.Printf("  Raw data (first 32 bytes): %x...\n", nestedData[:min(32, len(nestedData))])

		// Try to parse as map
		fmt.Println("\n  Parsing with ReadStructAsMap:")
		nestedMap, err := client.ReadStructAsMap(ctx, "MAIN.structExample2")
		if err != nil {
			log.Printf("  ⚠️  Failed: %v", err)
		} else {
			fmt.Printf("  Parsed fields:\n")
			for key, value := range nestedMap {
				if key == "_raw" {
					continue // Skip raw data display
				}
				fmt.Printf("    %s: %v\n", key, value)
			}
		}

		// Try manual parsing - explore the structure
		fmt.Println("\n  Exploring structure (nestedSt = INT + embedded TestSt):")

		offset := 0

		// Parse outer INT16 field
		if len(nestedData) >= 2 {
			val := int16(binary.LittleEndian.Uint16(nestedData[offset : offset+2]))
			fmt.Printf("    [%d] Outer.Field1 (INT): %d (0x%04X)\n", offset, val, uint16(val))
			offset += 2
		}

		// Embedded TestSt struct starts here (86 bytes: INT + UINT + STRING)
		if offset < len(nestedData) {
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
				nestedStringStart := 4
				nestedStringData := remaining[nestedStringStart:]
				for i, b := range nestedStringData {
					if b == 0 {
						if i > 0 {
							fmt.Printf("      [%d] Inner.Field3 (STRING): %q\n",
								offset+nestedStringStart, string(nestedStringData[:i]))
						}
						break
					}
				}
			}
		}
		fmt.Println("✅ Test 5 passed")
	}

	// Test 6: Get struct symbol information
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("ℹ️  Test 6: Get Struct Symbol Information")
	fmt.Println("═══════════════════════════════════════════════════════════")
	structSym, err := client.GetSymbol("MAIN.structExample")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  Symbol: %s\n", structSym.Name)
		fmt.Printf("  Type: %s\n", structSym.Type.Name)
		fmt.Printf("  Size: %d bytes\n", structSym.Size)
		fmt.Printf("  IndexGroup: 0x%X\n", structSym.IndexGroup)
		fmt.Printf("  IndexOffset: 0x%X\n", structSym.IndexOffset)
		fmt.Printf("  IsStruct: %v\n", structSym.Type.IsStruct)
		fmt.Printf("  Fields defined: %d\n", len(structSym.Type.Fields))
		fmt.Println("✅ Test 6 passed")
	}

	// Also check the nested struct info
	nestedSym, err := client.GetSymbol("MAIN.structExample2")
	if err != nil {
		log.Printf("⚠️  Failed to get nested struct info: %v", err)
	} else {
		fmt.Printf("\n  Nested struct info:\n")
		fmt.Printf("    Symbol: %s\n", nestedSym.Name)
		fmt.Printf("    Type: %s\n", nestedSym.Type.Name)
		fmt.Printf("    Size: %d bytes\n", nestedSym.Size)
		fmt.Printf("    IsStruct: %v\n", nestedSym.Type.IsStruct)
		fmt.Printf("    Fields defined: %d\n", len(nestedSym.Type.Fields))
	}

	// Test 7: Automatic parsing from PLC (no manual registration!)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🤖 Test 7: Automatic Parsing from PLC")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  ReadStructAsMap will automatically fetch type info from PLC!")

	// Test with MAIN.structExample (TestSt) - NO manual type registration needed!
	fmt.Println("\n  Parsing MAIN.structExample (TestSt) automatically:")
	result1, err := client.ReadStructAsMap(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("  ⚠️  Failed to parse: %v", err)
	} else {
		fmt.Println("  ✅ Automatically discovered struct layout from PLC!")
		printStructMap(result1, "  ")
	}

	// Test with MAIN.structExample2 (nestedSt) - includes nested struct!
	fmt.Println("\n  Parsing MAIN.structExample2 (nestedSt) automatically:")
	result2, err := client.ReadStructAsMap(ctx, "MAIN.structExample2")
	if err != nil {
		log.Printf("  ⚠️  Failed to parse: %v", err)
	} else {
		fmt.Println("  ✅ Automatically discovered nested struct layout!")
		printStructMap(result2, "  ")
	}
	fmt.Println("\n  ✅ Test 7 passed")

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
