package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

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
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create LocalDA instance with custom maxBlobSize if provided
	var opts []func(*LocalDA) *LocalDA
	if maxBlobSize != DefaultMaxBlobSize {
		opts = append(opts, WithMaxBlobSize(maxBlobSize))
	}
	da := NewLocalDA(logger, opts...)

	srv := proxy.NewServer(logger, host, port, da)
	logger.Info("Listening on", zap.String("host", host), zap.String("port", port), zap.Uint64("maxBlobSize", maxBlobSize))
	if err := srv.Start(context.Background()); err != nil {
		logger.Error("error while serving", zap.Error(err))
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT)
	<-interrupt
	fmt.Println("\nCtrl+C pressed. Exiting...")
	os.Exit(0)
}
