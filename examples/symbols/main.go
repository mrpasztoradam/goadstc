package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

// SymbolExport represents complete symbol information for JSON export
type SymbolExport struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Size        uint32      `json:"size"`
	IndexGroup  uint32      `json:"indexGroup"`
	IndexOffset uint32      `json:"indexOffset"`
	Comment     string      `json:"comment,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	RawData     string      `json:"rawData,omitempty"` // Hex encoded
}

func main() {
	// Parse command line flags
	exportFile := flag.String("export", "", "Export complete symbol table with values to JSON file")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Symbol Resolution Example (Milestone 1 & 2)          ║")
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

	// Test 1: Get symbol upload info
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📊 Test 1: Get Symbol Upload Info")
	fmt.Println("═══════════════════════════════════════════════════════════")
	symbolCount, symbolLength, err := client.GetSymbolUploadInfo(ctx)
	if err != nil {
		log.Fatalf("❌ Failed: %v", err)
	}
	fmt.Printf("  Symbol Count: %d\n", symbolCount)
	fmt.Printf("  Symbol Data Size: %d bytes\n", symbolLength)
	fmt.Println("✅ Test 1 passed")

	// Test 2: Refresh symbols (Milestone 2)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔄 Test 2: Refresh Symbol Table (Milestone 2)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := client.RefreshSymbols(ctx); err != nil {
		log.Fatalf("❌ Failed: %v", err)
	}
	fmt.Println("  ✅ Symbol table parsed and cached")

	allSymbols, err := client.ListSymbols(ctx)
	if err != nil {
		log.Fatalf("❌ Failed: %v", err)
	}
	fmt.Printf("  Parsed %d symbols\n", len(allSymbols))

	fmt.Println("  Sample symbols:")
	for i, sym := range allSymbols {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(allSymbols)-5)
			break
		}
		fmt.Printf("    - %s (%s, %d bytes)\n", sym.Name, sym.Type.Name, sym.Size)
	}
	fmt.Println("✅ Test 2 passed")

	// Test 3: Get symbol handle
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 3: Get Symbol Handle by Name")
	fmt.Println("═══════════════════════════════════════════════════════════")
	symbolName := "MAIN.uUint"
	fmt.Printf("  Looking up symbol: %q...\n", symbolName)
	handle, err := client.GetSymbolHandle(ctx, symbolName)
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  ✅ Symbol handle: 0x%08X\n", handle)

		if err := client.ReleaseSymbolHandle(ctx, handle); err != nil {
			log.Printf("❌ Failed to release: %v", err)
		} else {
			fmt.Println("  ✅ Handle released")
		}
	}

	// Test 4: Read symbol by name
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 4: Read Symbol by Name (Milestone 2)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	data, err := client.ReadSymbol(ctx, symbolName)
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  Read %d bytes from %s\n", len(data), symbolName)
		if len(data) == 2 {
			value := binary.LittleEndian.Uint16(data)
			fmt.Printf("  Value: %d (0x%04X)\n", value, value)
		}
		fmt.Println("✅ Test 4 passed")
	}

	// Test 5: Find symbols
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 5: Find Symbols by Pattern")
	fmt.Println("═══════════════════════════════════════════════════════════")
	matches, err := client.FindSymbols(ctx, "MAIN")
	if err != nil {
		log.Printf("❌ Failed: %v", err)
	} else {
		fmt.Printf("  Found %d symbols matching 'MAIN'\n", len(matches))
		for i, sym := range matches {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(matches)-3)
				break
			}
			fmt.Printf("    - %s\n", sym.Name)
		}
		fmt.Println("✅ Test 5 passed")
	}

	// Export symbols to JSON if requested
	if *exportFile != "" {
		fmt.Println("\n═══════════════════════════════════════════════════════════")
		fmt.Println("💾 Exporting Symbol Table to JSON")
		fmt.Println("═══════════════════════════════════════════════════════════")
		if err := exportSymbolTable(ctx, client, *exportFile); err != nil {
			log.Printf("❌ Failed to export: %v", err)
		} else {
			fmt.Printf("  ✅ Exported to: %s\n", *exportFile)
		}
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           Milestones 1 & 2 Complete!                    ║")
	fmt.Println("║  Next: Milestone 3 - Type-safe operations               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	if *exportFile == "" {
		fmt.Println("💡 Tip: Use -export filename.json to export all symbols with values")
	}
}

func exportSymbolTable(ctx context.Context, client *goadstc.Client, filename string) error {
	// Get all symbols
	symbols, err := client.ListSymbols(ctx)
	if err != nil {
		return fmt.Errorf("list symbols: %w", err)
	}

	exports := make([]SymbolExport, 0, len(symbols))

	fmt.Printf("  Reading values for %d symbols...\n", len(symbols))
	for i, sym := range symbols {
		if (i+1)%5 == 0 || i+1 == len(symbols) {
			fmt.Printf("  Progress: %d/%d\r", i+1, len(symbols))
		}

		export := SymbolExport{
			Name:        sym.Name,
			Type:        sym.Type.Name,
			Size:        sym.Size,
			IndexGroup:  sym.IndexGroup,
			IndexOffset: sym.IndexOffset,
			Comment:     sym.Comment,
		}

		// Try to read the value
		data, err := client.ReadSymbol(ctx, sym.Name)
		if err == nil && len(data) > 0 {
			export.RawData = fmt.Sprintf("%x", data)
			export.Value = parseValue(data, sym.Type.Name, sym.Size)
		}

		exports = append(exports, export)
	}
	fmt.Println()

	// Write to JSON file
	jsonData, err := json.MarshalIndent(exports, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func parseValue(data []byte, typeName string, size uint32) interface{} {
	if len(data) == 0 {
		return nil
	}

	// Parse common types
	switch typeName {
	case "BOOL":
		if len(data) >= 1 {
			return data[0] != 0
		}
	case "SINT", "INT8":
		if len(data) >= 1 {
			return int8(data[0])
		}
	case "USINT", "BYTE", "UINT8":
		if len(data) >= 1 {
			return uint8(data[0])
		}
	case "INT", "INT16":
		if len(data) >= 2 {
			return int16(binary.LittleEndian.Uint16(data))
		}
	case "UINT", "WORD", "UINT16":
		if len(data) >= 2 {
			return binary.LittleEndian.Uint16(data)
		}
	case "DINT", "INT32":
		if len(data) >= 4 {
			return int32(binary.LittleEndian.Uint32(data))
		}
	case "UDINT", "DWORD", "UINT32":
		if len(data) >= 4 {
			return binary.LittleEndian.Uint32(data)
		}
	case "LINT", "INT64":
		if len(data) >= 8 {
			return int64(binary.LittleEndian.Uint64(data))
		}
	case "ULINT", "LWORD", "UINT64":
		if len(data) >= 8 {
			return binary.LittleEndian.Uint64(data)
		}
	case "REAL", "FLOAT", "REAL32":
		if len(data) >= 4 {
			bits := binary.LittleEndian.Uint32(data)
			return fmt.Sprintf("%.6f", float32(bits))
		}
	case "LREAL", "DOUBLE", "REAL64":
		if len(data) >= 8 {
			bits := binary.LittleEndian.Uint64(data)
			return fmt.Sprintf("%.6f", float64(bits))
		}
	}

	// For complex types or unknown types, return hex string
	return fmt.Sprintf("0x%x", data)
}
