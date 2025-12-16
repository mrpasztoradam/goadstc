package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrpasztoradam/goadstc/middleware"
)

func main() {
	// Command line flags
	configFile := flag.String("config", "config.yaml", "Configuration file path")
	generateConfig := flag.Bool("generate-config", false, "Generate example configuration file and exit")
	flag.Parse()

	// Generate example config if requested
	if *generateConfig {
		if err := middleware.SaveExample("config.example.yaml"); err != nil {
			log.Fatalf("Failed to generate example config: %v", err)
		}
		log.Println("Generated config.example.yaml")
		return
	}

	// Load configuration
	config, err := middleware.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("╔═══════════════════════════════════════════════════════════╗")
	log.Println("║    GoADS HTTP/WebSocket Middleware Server                ║")
	log.Println("╚═══════════════════════════════════════════════════════════╝")
	log.Println()

	// Create server
	server, err := middleware.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server exited cleanly")
}
