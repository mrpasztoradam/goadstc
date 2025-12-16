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
	exportFile := flag.String("export", "", "Export complete symbol table with values to JSON file")
	flag.Parse()

	printHeader()
	client, ctx := setupClient()
	defer client.Close()

	runTest1GetSymbolUploadInfo(ctx, client)
	runTest2RefreshSymbols(ctx, client)
	runTest3GetSymbolHandle(ctx, client)
	runTest4ReadSymbolByName(ctx, client)
	runTest5FindSymbols(ctx, client)

	if *exportFile != "" {
		performExport(ctx, client, *exportFile)
	}

	printFooter(*exportFile)
}

func printHeader() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Symbol Resolution Example (Milestone 1 & 2)          ║")
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

func runTest1GetSymbolUploadInfo(ctx context.Context, client *goadstc.Client) {
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
}

func runTest2RefreshSymbols(ctx context.Context, client *goadstc.Client) {
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

}

func runTest3GetSymbolHandle(ctx context.Context, client *goadstc.Client) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 3: Get Symbol Handle by Name")
	fmt.Println("═══════════════════════════════════════════════════════════")
	symbolName := "MAIN.uUint"
	fmt.Printf("  Looking up symbol: %q...\n", symbolName)
	handle, err := client.GetSymbolHandle(ctx, symbolName)
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  ✅ Symbol handle: 0x%08X\n", handle)

	if err := client.ReleaseSymbolHandle(ctx, handle); err != nil {
		log.Printf("❌ Failed to release: %v", err)
	} else {
		fmt.Println("  ✅ Handle released")
	}
}

func runTest4ReadSymbolByName(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 4: Read Symbol by Name (Milestone 2)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	symbolName := "MAIN.uUint"
	data, err := client.ReadSymbol(ctx, symbolName)
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
		return
	}
	fmt.Printf("  Read %d bytes from %s\n", len(data), symbolName)
	if len(data) == 2 {
		value := binary.LittleEndian.Uint16(data)
		fmt.Printf("  Value: %d (0x%04X)\n", value, value)
	}
	fmt.Println("✅ Test 4 passed")
}

func runTest5FindSymbols(ctx context.Context, client *goadstc.Client) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("🔍 Test 5: Find Symbols by Pattern")
	fmt.Println("═══════════════════════════════════════════════════════════")
	matches, err := client.FindSymbols(ctx, "MAIN")
	if err != nil {
		log.Printf("❌ Failed: %v", err)
		return
	}
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

func performExport(ctx context.Context, client *goadstc.Client, exportFile string) {
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("💾 Exporting Symbol Table to JSON")
	fmt.Println("═══════════════════════════════════════════════════════════")
	if err := exportSymbolTable(ctx, client, exportFile); err != nil {
		log.Printf("❌ Failed to export: %v", err)
	} else {
		fmt.Printf("  ✅ Exported to: %s\n", exportFile)
	}
}

func printFooter(exportFile string) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           Milestones 1 & 2 Complete!                    ║")
	fmt.Println("║  Next: Milestone 3 - Type-safe operations               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	if exportFile == "" {
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

	if value := parse8BitTypes(data, typeName); value != nil {
		return value
	}
	if value := parse16BitTypes(data, typeName); value != nil {
		return value
	}
	if value := parse32BitTypes(data, typeName); value != nil {
		return value
	}
	if value := parse64BitTypes(data, typeName); value != nil {
		return value
	}

	return fmt.Sprintf("0x%x", data)
}

func parse8BitTypes(data []byte, typeName string) interface{} {
	if len(data) < 1 {
		return nil
	}
	switch typeName {
	case "BOOL":
		return data[0] != 0
	case "SINT", "INT8":
		return int8(data[0])
	case "USINT", "BYTE", "UINT8":
		return uint8(data[0])
	}
	return nil
}

func parse16BitTypes(data []byte, typeName string) interface{} {
	if len(data) < 2 {
		return nil
	}
	switch typeName {
	case "INT", "INT16":
		return int16(binary.LittleEndian.Uint16(data))
	case "UINT", "WORD", "UINT16":
		return binary.LittleEndian.Uint16(data)
	}
	return nil
}

func parse32BitTypes(data []byte, typeName string) interface{} {
	if len(data) < 4 {
		return nil
	}
	switch typeName {
	case "DINT", "INT32":
		return int32(binary.LittleEndian.Uint32(data))
	case "UDINT", "DWORD", "UINT32":
		return binary.LittleEndian.Uint32(data)
	case "REAL", "FLOAT", "REAL32":
		bits := binary.LittleEndian.Uint32(data)
		return fmt.Sprintf("%.6f", float32(bits))
	}
	return nil
}

func parse64BitTypes(data []byte, typeName string) interface{} {
	if len(data) < 8 {
		return nil
	}
	switch typeName {
	case "LINT", "INT64":
		return int64(binary.LittleEndian.Uint64(data))
	case "ULINT", "LWORD", "UINT64":
		return binary.LittleEndian.Uint64(data)
	case "LREAL", "DOUBLE", "REAL64":
		bits := binary.LittleEndian.Uint64(data)
		return fmt.Sprintf("%.6f", float64(bits))
	}
	return nil
}
