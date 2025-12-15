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

	// Test 1: TIME operations (duration in milliseconds)
	fmt.Println("\n=== Test 1: TIME operations ===")

	// Write a duration of 5 seconds 250 milliseconds
	duration := 5*time.Second + 250*time.Millisecond
	if err := client.WriteTime(ctx, "MAIN.tTime", duration); err != nil {
		log.Fatalf("Failed to write TIME: %v", err)
	}
	fmt.Printf("Wrote TIME: %v\n", duration)

	// Read it back
	readDuration, err := client.ReadTime(ctx, "MAIN.tTime")
	if err != nil {
		log.Fatalf("Failed to read TIME: %v", err)
	}
	fmt.Printf("Read TIME: %v\n", readDuration)

	// Test 2: DATE operations (date as Unix timestamp)
	fmt.Println("\n=== Test 2: DATE operations ===")

	// Write a date (January 15, 2025)
	dateValue := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if err := client.WriteDate(ctx, "MAIN.dDate", dateValue); err != nil {
		log.Fatalf("Failed to write DATE: %v", err)
	}
	fmt.Printf("Wrote DATE: %v\n", dateValue.Format("2006-01-02"))

	// Read it back
	readDate, err := client.ReadDate(ctx, "MAIN.dDate")
	if err != nil {
		log.Fatalf("Failed to read DATE: %v", err)
	}
	fmt.Printf("Read DATE: %v\n", readDate.Format("2006-01-02"))

	// Test 3: TIME_OF_DAY operations (milliseconds since midnight)
	fmt.Println("\n=== Test 3: TIME_OF_DAY operations ===")

	// Write time of day: 14:30:15.500 (2:30 PM and 15.5 seconds)
	tod := 14*time.Hour + 30*time.Minute + 15*time.Second + 500*time.Millisecond
	if err := client.WriteTimeOfDay(ctx, "MAIN.todTimeOfDay", tod); err != nil {
		log.Fatalf("Failed to write TIME_OF_DAY: %v", err)
	}
	fmt.Printf("Wrote TIME_OF_DAY: %v\n", tod)

	// Read it back
	readTOD, err := client.ReadTimeOfDay(ctx, "MAIN.todTimeOfDay")
	if err != nil {
		log.Fatalf("Failed to read TIME_OF_DAY: %v", err)
	}
	fmt.Printf("Read TIME_OF_DAY: %v\n", readTOD)

	// Convert to human-readable format
	hours := int(readTOD.Hours())
	minutes := int(readTOD.Minutes()) % 60
	seconds := int(readTOD.Seconds()) % 60
	millis := int(readTOD.Milliseconds()) % 1000
	fmt.Printf("As time: %02d:%02d:%02d.%03d\n", hours, minutes, seconds, millis)

	// Test 4: DATE_AND_TIME operations (full timestamp)
	fmt.Println("\n=== Test 4: DATE_AND_TIME operations ===")

	// Write a timestamp (December 15, 2025, 16:45:30 UTC)
	timestamp := time.Date(2025, 12, 15, 16, 45, 30, 0, time.UTC)
	if err := client.WriteDateAndTime(ctx, "MAIN.dtDateAndTime", timestamp); err != nil {
		log.Fatalf("Failed to write DATE_AND_TIME: %v", err)
	}
	fmt.Printf("Wrote DATE_AND_TIME: %v\n", timestamp.Format("2006-01-02 15:04:05"))

	// Read it back
	readDT, err := client.ReadDateAndTime(ctx, "MAIN.dtDateAndTime")
	if err != nil {
		log.Fatalf("Failed to read DATE_AND_TIME: %v", err)
	}
	fmt.Printf("Read DATE_AND_TIME: %v\n", readDT.Format("2006-01-02 15:04:05"))

	// Test 5: WSTRING operations (Unicode strings)
	fmt.Println("\n=== Test 5: WSTRING operations ===")

	// Write a Unicode string with special characters
	unicodeText := "Hello 世界 🌍!"
	if err := client.WriteWString(ctx, "MAIN.wWstring", unicodeText); err != nil {
		log.Fatalf("Failed to write WSTRING: %v", err)
	}
	fmt.Printf("Wrote WSTRING: %s\n", unicodeText)

	// Read it back
	readWString, err := client.ReadWString(ctx, "MAIN.wWstring")
	if err != nil {
		log.Fatalf("Failed to read WSTRING: %v", err)
	}
	fmt.Printf("Read WSTRING: %s\n", readWString)

	fmt.Println("\n=== All tests completed successfully! ===")
}
