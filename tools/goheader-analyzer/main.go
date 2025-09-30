package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/celestiaorg/go-header/local"
	goheaderstore "github.com/celestiaorg/go-header/store"
	"github.com/evstack/ev-node/types"
	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	badger4 "github.com/ipfs/go-ds-badger4"
)

const (
	evPrefix = "0"

	headerSync = "headerSync"
	dataSync   = "dataSync"
)

var timeout = 30 * time.Second

func newPrefixKV(kvStore ds.Batching, prefix string) ds.Batching {
	return ktds.Wrap(kvStore, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)})
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: goheader-analyzer <dir>")
		os.Exit(1)
	}

	datastore, err := badger4.NewDatastore(os.Args[1], nil)
	if err != nil {
		log.Fatal(err)
	}

	ss, err := goheaderstore.NewStore[*types.SignedHeader](
		newPrefixKV(datastore, evPrefix),
		goheaderstore.WithStorePrefix(headerSync),
		goheaderstore.WithMetrics(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	storeHead, err := ss.Head(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Store Head:", storeHead)

	storeTail, err := ss.Tail(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Store Tail:", storeTail)

	localExchange := local.NewExchange(ss)

	head, err := localExchange.Head(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Exchanger Head:", head)

	fmt.Println("Trying to get height from Exchanger: 1")
	height1, err := localExchange.GetByHeight(ctx, 1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Height 1:", height1)
}
