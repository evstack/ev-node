package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	proxy "github.com/evstack/ev-node/da/jsonrpc"
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

	srv := proxy.NewServer(logger, host, port, da)
	logger.Info().Str("host", host).Str("port", port).Uint64("maxBlobSize", maxBlobSize).Msg("Listening on")
	if err := srv.Start(context.Background()); err != nil {
		logger.Error().Err(err).Msg("error while serving")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT)
	<-interrupt
	fmt.Println("\nCtrl+C pressed. Exiting...")
	os.Exit(0)
}
