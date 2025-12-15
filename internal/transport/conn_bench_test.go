package transport_test

import (
	"context"
	"testing"
	"time"

	"github.com/mrpasztoradam/goadstc/internal/ams"
	"github.com/mrpasztoradam/goadstc/internal/transport"
)

// BenchmarkConnectionCreation measures connection establishment overhead
func BenchmarkConnectionCreation(b *testing.B) {
	// Skip if no PLC available
	b.Skip("Requires PLC connection - run manually with real PLC")

	address := "192.168.1.100:48898"
	timeout := 5 * time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		conn, err := transport.Dial(ctx, address, timeout)
		if err != nil {
			b.Fatalf("Failed to dial: %v", err)
		}
		conn.Close()
	}
}

// BenchmarkRequestLatency measures round-trip request latency
func BenchmarkRequestLatency(b *testing.B) {
	// Skip if no PLC available
	b.Skip("Requires PLC connection - run manually with real PLC")

	address := "192.168.1.100:48898"
	timeout := 5 * time.Second
	ctx := context.Background()

	conn, err := transport.Dial(ctx, address, timeout)
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Create a simple ReadState request
	targetNetID := ams.NetID{192, 168, 1, 100, 1, 1}
	sourceNetID := ams.NetID{0, 0, 0, 0, 0, 0}
	reqData := make([]byte, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invokeID := conn.NextInvokeID()
		packet := ams.NewRequestPacket(
			targetNetID, 851,
			sourceNetID, 32905,
			uint16(0x0001), // ReadDeviceInfo command
			invokeID,
			reqData,
		)

		_, err := conn.SendRequest(ctx, packet)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkConcurrentRequests measures performance under concurrent load
func BenchmarkConcurrentRequests(b *testing.B) {
	// Skip if no PLC available
	b.Skip("Requires PLC connection - run manually with real PLC")

	address := "192.168.1.100:48898"
	timeout := 5 * time.Second
	ctx := context.Background()

	conn, err := transport.Dial(ctx, address, timeout)
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	targetNetID := ams.NetID{192, 168, 1, 100, 1, 1}
	sourceNetID := ams.NetID{0, 0, 0, 0, 0, 0}
	reqData := make([]byte, 0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invokeID := conn.NextInvokeID()
			packet := ams.NewRequestPacket(
				targetNetID, 851,
				sourceNetID, 32905,
				uint16(0x0001),
				invokeID,
				reqData,
			)

			_, err := conn.SendRequest(ctx, packet)
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
		}
	})
}
