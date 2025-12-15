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

	// Test 5: Get struct symbol information
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("ℹ️  Test 5: Get Struct Symbol Information")
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
		fmt.Println("✅ Test 5 passed")
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 4 Notes                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📝 Struct Field Access:")
	fmt.Println("  TwinCAT 3 does not export individual struct fields as symbols.")
	fmt.Println("  To access struct fields, you have two options:")
	fmt.Println()
	fmt.Println("  1. Read entire struct and parse manually:")
	fmt.Println("     data := client.ReadSymbol(ctx, \"MAIN.myStruct\")")
	fmt.Println("     field1 := binary.LittleEndian.Uint16(data[offset:])")
	fmt.Println()
	fmt.Println("  2. Use direct index read with calculated offset:")
	fmt.Println("     // Requires knowing field offset in advance")
	fmt.Println("     client.Read(ctx, indexGroup, indexOffset + fieldOffset, size)")
	fmt.Println()
	fmt.Println("  For production use, consider:")
	fmt.Println("  • Generating Go structs from TwinCAT type definitions")
	fmt.Println("  • Using code generation tools")
	fmt.Println("  • Documenting struct layouts in your PLC project")
	fmt.Println()
	fmt.Println("  The type-safe methods work great with known offsets:")
	fmt.Println("  • ReadInt16/WriteInt16 at specific index group/offset")
	fmt.Println("  • ReadUint32/WriteUint32 for struct fields")
	fmt.Println("  • Manual offset calculation based on TwinCAT struct definition")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
