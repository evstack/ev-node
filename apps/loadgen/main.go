package main

import (
	"log"
	"os"

	"github.com/evstack/ev-node/apps/loadgen/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	log.SetOutput(os.Stdout)
}
