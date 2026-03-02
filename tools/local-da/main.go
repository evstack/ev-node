package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

const (
	defaultHost = "localhost"
	defaultPort = "7980"
)

func main() {
	var (
		host        string
		port        string
		listenAll   bool
		maxBlobSize uint64
		blockTime   time.Duration
	)
	flag.StringVar(&port, "port", defaultPort, "listening port")
	flag.StringVar(&host, "host", defaultHost, "listening address")
	flag.BoolVar(&listenAll, "listen-all", false, "listen on all network interfaces (0.0.0.0) instead of just localhost")
	flag.Uint64Var(&maxBlobSize, "max-blob-size", DefaultMaxBlobSize, "maximum blob size in bytes")
	flag.DurationVar(&blockTime, "block-time", DefaultBlockTime, "time between empty blocks (e.g., 1s, 500ms)")
	flag.Parse()

	if listenAll {
		host = "0.0.0.0"
	}

	// create logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Str("component", "da").Logger()

	// Create LocalDA instance with custom maxBlobSize if provided
	var opts []func(*LocalDA) *LocalDA
	if maxBlobSize != DefaultMaxBlobSize {
		opts = append(opts, WithMaxBlobSize(maxBlobSize))
	}
	if blockTime != DefaultBlockTime {
		opts = append(opts, WithBlockTime(blockTime))
	}
	da := NewLocalDA(logger, opts...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	da.Start(ctx)

	addr := fmt.Sprintf("%s:%s", host, port)
	srv, err := startBlobServer(logger, addr, da)
	if err != nil {
		logger.Error().Err(err).Msg("error while creating blob RPC server")
		os.Exit(1)
	}

	logger.Info().Str("host", host).Str("port", port).Uint64("maxBlobSize", maxBlobSize).Dur("blockTime", blockTime).Msg("Listening on")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT)
	<-interrupt
	fmt.Println("\nCtrl+C pressed. Exiting...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("error shutting down server")
	}
	os.Exit(0)
}
