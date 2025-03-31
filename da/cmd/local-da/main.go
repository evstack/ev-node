package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	coreda "github.com/rollkit/rollkit/core/da"
	proxy "github.com/rollkit/rollkit/da/proxy/jsonrpc"
)

const (
	defaultHost = "localhost"
	defaultPort = "7980"
)

func main() {
	var (
		host      string
		port      string
		listenAll bool
	)
	flag.StringVar(&port, "port", defaultPort, "listening port")
	flag.StringVar(&host, "host", defaultHost, "listening address")
	flag.BoolVar(&listenAll, "listen-all", false, "listen on all network interfaces (0.0.0.0) instead of just localhost")
	flag.Parse()

	if listenAll {
		host = "0.0.0.0"
	}

	srv := proxy.NewServer(host, port, coreda.NewDummyDA(100_000, 0, 0))
	log.Printf("Listening on: %s:%s", host, port)
	if err := srv.Start(context.Background()); err != nil {
		log.Fatal("error while serving:", err)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT)
	<-interrupt
	fmt.Println("\nCtrl+C pressed. Exiting...")
	os.Exit(0)
}
