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
	fmt.Println("✅ Connected successfully\n")

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
	fmt.Println("✅ Test 1 passed\n")

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
	fmt.Println("✅ Test 2 passed\n")

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
		if len(data) == 4 {
			value := binary.LittleEndian.Uint32(data)
			fmt.Printf("  Value: %d (0x%08X)\n", value, value)
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

	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           Milestones 1 & 2 Complete!                    ║")
	fmt.Println("║  Next: Milestone 3 - Type-safe operations               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
}
