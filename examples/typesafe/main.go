package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mrpasztoradam/goadstc"
	"github.com/mrpasztoradam/goadstc/internal/ams"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║    Type-Safe Operations Example (Milestone 3)           ║")
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
	fmt.Println()

	ctx := context.Background()

	// Test 1: Read UINT16
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 1: Read UINT16 (Type-Safe)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	uintVal, err := client.ReadUint16(ctx, "MAIN.uUint")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  MAIN.uUint = %d (0x%04X)\n", uintVal, uintVal)
		fmt.Println("✅ Test 1 passed")
	}

	// Test 2: Write UINT16
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 2: Write UINT16 (Type-Safe)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	newValue := uint16(100)
	fmt.Printf("  Writing %d to MAIN.uUint...\n", newValue)
	if err := client.WriteUint16(ctx, "MAIN.uUint", newValue); err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Println("  ✅ Write successful")

		// Verify the write
		verifyVal, err := client.ReadUint16(ctx, "MAIN.uUint")
		if err != nil {
			log.Printf("⚠️  Failed to verify: %v", err)
		} else {
			fmt.Printf("  Verification: MAIN.uUint = %d\n", verifyVal)
			if verifyVal == newValue {
				fmt.Println("✅ Test 2 passed")
			} else {
				fmt.Printf("❌ Mismatch: expected %d, got %d\n", newValue, verifyVal)
			}
		}
	}

	// Test 3: Read INT16
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 3: Read INT16 (Type-Safe)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	intVal, err := client.ReadInt16(ctx, "MAIN.iInt")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  MAIN.iInt = %d (0x%04X)\n", intVal, uint16(intVal))
		fmt.Println("✅ Test 3 passed")
	}

	// Test 4: Write INT16
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 4: Write INT16 (Type-Safe)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	newIntValue := int16(-42)
	fmt.Printf("  Writing %d to MAIN.iInt...\n", newIntValue)
	if err := client.WriteInt16(ctx, "MAIN.iInt", newIntValue); err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Println("  ✅ Write successful")

		// Verify the write
		verifyIntVal, err := client.ReadInt16(ctx, "MAIN.iInt")
		if err != nil {
			log.Printf("⚠️  Failed to verify: %v", err)
		} else {
			fmt.Printf("  Verification: MAIN.iInt = %d\n", verifyIntVal)
			if verifyIntVal == newIntValue {
				fmt.Println("✅ Test 4 passed")
			} else {
				fmt.Printf("❌ Mismatch: expected %d, got %d\n", newIntValue, verifyIntVal)
			}
		}
	}

	// Test 5: Read WORD (UINT16)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📖 Test 5: Read WORD (Type-Safe)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	wordVal, err := client.ReadUint16(ctx, "MAIN.wWord")
	if err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Printf("  MAIN.wWord = %d (0x%04X)\n", wordVal, wordVal)
		fmt.Println("✅ Test 5 passed")
	}

	// Test 6: Write WORD
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("✍️  Test 6: Write WORD (Type-Safe)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	newWordValue := uint16(0xABCD)
	fmt.Printf("  Writing 0x%04X to MAIN.wWord...\n", newWordValue)
	if err := client.WriteUint16(ctx, "MAIN.wWord", newWordValue); err != nil {
		log.Printf("⚠️  Failed: %v", err)
	} else {
		fmt.Println("  ✅ Write successful")

		// Verify the write
		verifyWordVal, err := client.ReadUint16(ctx, "MAIN.wWord")
		if err != nil {
			log.Printf("⚠️  Failed to verify: %v", err)
		} else {
			fmt.Printf("  Verification: MAIN.wWord = 0x%04X\n", verifyWordVal)
			if verifyWordVal == newWordValue {
				fmt.Println("✅ Test 6 passed")
			} else {
				fmt.Printf("❌ Mismatch: expected 0x%04X, got 0x%04X\n", newWordValue, verifyWordVal)
			}
		}
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Milestone 3 Complete!                       ║")
	fmt.Println("║  Type-safe operations provide cleaner, safer API         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎯 Benefits:")
	fmt.Println("  • No manual byte encoding/decoding")
	fmt.Println("  • Type safety at compile time")
	fmt.Println("  • Cleaner, more readable code")
	fmt.Println("  • Automatic endianness handling")
}
