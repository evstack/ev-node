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
	)
	flag.StringVar(&port, "port", defaultPort, "listening port")
	flag.StringVar(&host, "host", defaultHost, "listening address")
	flag.BoolVar(&listenAll, "listen-all", false, "listen on all network interfaces (0.0.0.0) instead of just localhost")
	flag.Uint64Var(&maxBlobSize, "max-blob-size", DefaultMaxBlobSize, "maximum blob size in bytes")
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
	da := NewLocalDA(logger, opts...)

	addr := fmt.Sprintf("%s:%s", host, port)
	srv, err := startBlobServer(logger, addr, da)
	if err != nil {
		logger.Error().Err(err).Msg("error while creating blob RPC server")
		os.Exit(1)
	}

	logger.Info().Str("host", host).Str("port", port).Uint64("maxBlobSize", maxBlobSize).Msg("Listening on")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT)
	<-interrupt
	fmt.Println("\nCtrl+C pressed. Exiting...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("error shutting down server")
	}
	os.Exit(0)
}
