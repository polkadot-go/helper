// main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/polkadot-go/helper/core"
	"github.com/polkadot-go/helper/core/config"

	// Import to trigger registrations
	_ "github.com/polkadot-go/helper/core/config"
	_ "github.com/polkadot-go/helper/data/mysql"
	_ "github.com/polkadot-go/helper/managers/network"
)

func main() {
	// Set config file if needed
	if len(os.Args) > 1 {
		config.SetConfigFile(os.Args[1])
	}

	// Initialize all components
	if err := core.Initialize(); err != nil {
		log.Fatal("Failed to initialize:", err)
	}

	log.Println("System initialized:", core.GetInitOrder())

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := core.Shutdown(ctx); err != nil {
		log.Fatal("Failed to shutdown:", err)
	}

	log.Println("Shutdown complete")
}
