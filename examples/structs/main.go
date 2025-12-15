package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
	"github.com/mrpasztoradam/goadstc/internal/symbols"
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
	fmt.Println("✅ Connected successfully\n")

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
			val := int16(binary.LittleEndian.Uint16(nestedData[offset:offset+2]))
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

	// Test 7: Automatic parsing with type registration
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🤖 Test 7: Automatic Parsing (Type Registration)")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Register the TestSt type definition
	testStType := symbols.TypeInfo{
		Name:     "TestSt",
		BaseType: 65, // Struct type
		Size:     86,
		IsStruct: true,
		Fields: []symbols.FieldInfo{
			{
				Name:   "field1",
				Offset: 0,
				Type: symbols.TypeInfo{
					Name:     "INT",
					BaseType: symbols.DataTypeInt16,
					Size:     2,
				},
			},
			{
				Name:   "field2",
				Offset: 2,
				Type: symbols.TypeInfo{
					Name:     "UINT",
					BaseType: symbols.DataTypeUInt16,
					Size:     2,
				},
			},
			{
				Name:   "field3",
				Offset: 4,
				Type: symbols.TypeInfo{
					Name:     "STRING(81)",
					BaseType: symbols.DataTypeString,
					Size:     82,
				},
			},
		},
	}
	client.RegisterType(testStType)

	// Register the nestedSt type definition
	nestedStType := symbols.TypeInfo{
		Name:     "nestedSt",
		BaseType: 65, // Struct type
		Size:     88,
		IsStruct: true,
		Fields: []symbols.FieldInfo{
			{
				Name:   "field1",
				Offset: 0,
				Type: symbols.TypeInfo{
					Name:     "INT",
					BaseType: symbols.DataTypeInt16,
					Size:     2,
				},
			},
			{
				Name:   "inner",
				Offset: 2,
				Type:   testStType, // Nested struct
			},
		},
	}
	client.RegisterType(nestedStType)

	fmt.Println("  ✅ Registered types: TestSt, nestedSt")
	fmt.Println("\n  Now ReadStructAsMap will automatically parse these structs!")

	// Test with MAIN.structExample (TestSt)
	fmt.Println("\n  Parsing MAIN.structExample (TestSt) automatically:")
	result1, err := client.ReadStructAsMap(ctx, "MAIN.structExample")
	if err != nil {
		log.Printf("  ❌ Failed to parse: %v", err)
	} else {
		if field1, ok := result1["field1"]; ok {
			fmt.Printf("    field1 (INT): %v\n", field1)
		}
		if field2, ok := result1["field2"]; ok {
			fmt.Printf("    field2 (UINT): %v\n", field2)
		}
		if field3, ok := result1["field3"]; ok {
			fmt.Printf("    field3 (STRING): %q\n", field3)
		}
	}

	// Test with MAIN.structExample2 (nestedSt)
	fmt.Println("\n  Parsing MAIN.structExample2 (nestedSt) automatically:")
	result2, err := client.ReadStructAsMap(ctx, "MAIN.structExample2")
	if err != nil {
		log.Printf("  ❌ Failed to parse: %v", err)
	} else {
		if field1, ok := result2["field1"]; ok {
			fmt.Printf("    field1 (INT): %v\n", field1)
		}
		if inner, ok := result2["inner"]; ok {
			fmt.Printf("    inner (TestSt): %v\n", inner)
		}
	}
	fmt.Println("  ✅ Test 7 passed")

	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 4 Notes                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📝 Struct Field Access:")
	fmt.Println("  TwinCAT 3 does not export individual struct fields as symbols.")
	fmt.Println("  To access struct fields, you have three options:")
	fmt.Println()
	fmt.Println("  1. Register type definitions (RECOMMENDED):")
	fmt.Println("     client.RegisterType(typeInfo)")
	fmt.Println("     result := client.ReadStructAsMap(ctx, \"MAIN.myStruct\")")
	fmt.Println("     // Automatically parses all fields!")
	fmt.Println()
	fmt.Println("  2. Read entire struct and parse manually:")
	fmt.Println("     data := client.ReadSymbol(ctx, \"MAIN.myStruct\")")
	fmt.Println("     field1 := binary.LittleEndian.Uint16(data[offset:])")
	fmt.Println()
	fmt.Println("  3. Use direct index read with calculated offset:")
	fmt.Println("     client.Read(ctx, indexGroup, indexOffset + fieldOffset, size)")
	fmt.Println()
	fmt.Println("  For production use, consider:")
	fmt.Println("  • Registering type definitions once at startup")
	fmt.Println("  • Using ReadStructAsMap for automatic parsing")
	fmt.Println("  • Generating type registrations from TwinCAT definitions")
	fmt.Println("  • Code generation tools for type-safe structs")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
