//go:build ignore
// +build ignore

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	plcIP := "10.10.0.3:48898"
	plcNetID := ams.NetID{10, 0, 10, 20, 1, 1}
	pcNetID := ams.NetID{10, 10, 0, 10, 1, 1}

	fmt.Println("Creating client with debug logging...")
	client, err := goadstc.New(
		goadstc.WithTarget(plcIP),
		goadstc.WithAMSNetID(plcNetID),
		goadstc.WithSourceNetID(pcNetID),
		goadstc.WithAMSPort(851),
		goadstc.WithTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	fmt.Println("Connected successfully\n")

	ctx := context.Background()

	// Read original struct data
	fmt.Println("1. Reading original struct...")
	originalData, err := client.ReadSymbol(ctx, "MAIN.structExample")
	if err != nil {
		log.Fatalf("Failed to read struct: %v", err)
	}
	fmt.Printf("Original struct data (%d bytes):\n%s\n\n", len(originalData), hex.Dump(originalData))

	// Read original as map
	fmt.Println("2. Reading original struct as map...")
	originalMap, err := client.ReadSymbolValue(ctx, "MAIN.structExample")
	if err != nil {
		log.Fatalf("Failed to read struct value: %v", err)
	}
	fmt.Printf("Original values: %+v\n\n", originalMap)

	// Try to write using WriteStructFields
	fmt.Println("3. Writing fields using WriteStructFields...")
	err = client.WriteStructFields(ctx, "MAIN.structExample", map[string]interface{}{
		"iTest":  int16(12345),
		"uiTest": uint16(54321),
	})
	if err != nil {
		log.Fatalf("Failed to write struct fields: %v", err)
	}
	fmt.Println("Write reported success\n")

	// Read back immediately
	fmt.Println("4. Reading struct back immediately...")
	newData, err := client.ReadSymbol(ctx, "MAIN.structExample")
	if err != nil {
		log.Fatalf("Failed to read struct: %v", err)
	}
	fmt.Printf("New struct data (%d bytes):\n%s\n\n", len(newData), hex.Dump(newData))

	// Read as map to see values
	fmt.Println("5. Reading struct as map to see values...")
	newMap, err := client.ReadSymbolValue(ctx, "MAIN.structExample")
	if err != nil {
		log.Fatalf("Failed to read struct value: %v", err)
	}
	fmt.Printf("New values: %+v\n\n", newMap)

	// Compare
	if string(originalData) == string(newData) {
		fmt.Println("❌ PROBLEM: Data unchanged - write did not persist!")
	} else {
		fmt.Println("✅ Data changed - write persisted!")
	}
}
