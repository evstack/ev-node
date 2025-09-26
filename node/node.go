package node

import (
	"github.com/evstack/ev-node/pkg/p2p/key"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
	coreda "github.com/evstack/ev-node/core/da"
	coreexecutor "github.com/evstack/ev-node/core/execution"
	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/service"
	"github.com/evstack/ev-node/pkg/signer"
)

// Node is the interface for an application node
type Node interface {
	service.Service

	IsRunning() bool
}

type NodeOptions struct {
	BlockOptions block.BlockOptions
}

// NewNode returns a new Full or Light Node based on the config
// This is the entry point for composing a node, when compiling a node, you need to provide an executor
// Example executors can be found in apps/
func NewNode(
	conf config.Config,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	da coreda.DA,
	signer signer.Signer,
	nodeKey *key.NodeKey,
	genesis genesis.Genesis,
	database ds.Batching,
	metricsProvider MetricsProvider,
	logger zerolog.Logger,
	nodeOptions NodeOptions,
) (Node, error) {
	if conf.Node.Light {
		panic("todo: not impl")
		//return newLightNode(conf, genesis, nodeKey, database, logger)
	}

	if err := nodeOptions.BlockOptions.Validate(); err != nil {
		nodeOptions.BlockOptions = block.DefaultBlockOptions()
	}

	return newFullNode(
		conf,
		nodeKey,
		signer,
		genesis,
		database,
		exec,
		sequencer,
		da,
		metricsProvider,
		logger,
		nodeOptions,
	)
}
