package example

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/rpc/client"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/rs/zerolog"
)

// StartStoreServer starts a Store RPC server with the provided store instance
func StartStoreServer(s store.Store, address string, logger zerolog.Logger) {
	// Create and start the server
	// Start RPC server
	rpcAddr := fmt.Sprintf("%s:%d", "localhost", 8080)

	cfg := config.DefaultConfig()
	handler, err := server.NewServiceHandler(s, nil, nil, logger, cfg, nil)
	if err != nil {
		panic(err)
	}

	rpcServer := &http.Server{
		Addr:         rpcAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start the server in a separate goroutine
	go func() {
		if err := rpcServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("RPC server error")
			os.Exit(1)
		}
	}()
}

// ExampleClient demonstrates how to use the Store RPC client
func ExampleClient() {
	// Create a new client
	client := client.NewClient("http://localhost:8080")
	ctx := context.Background()

	// Get the current state
	state, err := client.GetState(ctx)
	if err != nil {
		log.Fatalf("Failed to get state: %v", err)
	}
	log.Printf("Current state: %+v", state)

	// Get metadata
	metadataKey := "example_key"
	metadataValue, err := client.GetMetadata(ctx, metadataKey)
	if err != nil {
		log.Printf("Metadata not found: %v", err)
	} else {
		log.Printf("Metadata value: %s", string(metadataValue))
	}

	// Get a block by height
	height := uint64(10)
	block, err := client.GetBlockByHeight(ctx, height)
	if err != nil {
		log.Fatalf("Failed to get block: %v", err)
	}
	log.Printf("Block at height %d: %+v", height, block)
}

// ExampleServer demonstrates how to create and start a Store RPC server
func ExampleServer(s store.Store) {
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("component", "exampleServer").Logger()

	// Start RPC server
	rpcAddr := fmt.Sprintf("%s:%d", "localhost", 8080)
	cfg := config.DefaultConfig()
	handler, err := server.NewServiceHandler(s, nil, nil, logger, cfg, nil)
	if err != nil {
		panic(err)
	}

	rpcServer := &http.Server{
		Addr:         rpcAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start the server in a separate goroutine
	go func() {
		if err := rpcServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("RPC server error: %v", err)
		}
	}()

	log.Println("Store RPC server started on localhost:8080")
	// The server will continue running until the program exits
}
