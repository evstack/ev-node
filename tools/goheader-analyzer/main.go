package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/celestiaorg/go-header/local"
	goheaderstore "github.com/celestiaorg/go-header/store"
	"github.com/evstack/ev-node/node"
	"github.com/evstack/ev-node/types"
	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	badger4 "github.com/ipfs/go-ds-badger4"
)

const (
	headerSync = "headerSync"
	dataSync   = "dataSync"
)

var timeout = 30 * time.Second

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: goheader-analyzer <dir>")
		os.Exit(1)
	}

	datastore, err := badger4.NewDatastore(os.Args[1], nil)
	if err != nil {
		log.Fatalf("Failed to load db: %s", err)
	}

	evolveDB := ktds.Wrap(datastore, &ktds.PrefixTransform{
		Prefix: ds.NewKey(node.EvPrefix),
	})

	ss, err := goheaderstore.NewStore[*types.SignedHeader](
		evolveDB,
		goheaderstore.WithStorePrefix(headerSync),
	)

	ctx := context.Background()

	if err := ss.Start(ctx); err != nil {
		log.Fatalf("Failed to start goheader store: %s", err)
	}
	defer ss.Stop(ctx)

	storeHead, err := ss.Head(ctx)
	if err != nil {
		log.Fatalf("Failed to get store head: %s", err)
	}
	fmt.Println("Store Head:", storeHead)

	localExchange := local.NewExchange(ss)

	head, err := localExchange.Head(ctx)
	if err != nil {
		log.Fatalf("Failed to get exchanger head: %s", err)
	}
	fmt.Println("Exchanger Head:", head)

	fmt.Println("Trying to get height from Exchanger: 1")
	height1, err := localExchange.GetByHeight(ctx, 1)
	if err != nil {
		log.Fatalf("Failed to get exchanger height: %s", err)
	}
	fmt.Println("Height 1:", height1)
}
