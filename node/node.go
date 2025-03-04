package node

import (
	"context"

	"cosmossdk.io/log"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/rollkit/rollkit/config"
	coreexecutor "github.com/rollkit/rollkit/core/execution"
	coresequencer "github.com/rollkit/rollkit/core/sequencer"
	"github.com/rollkit/rollkit/types"
)

// Node is the interface for a rollup node
type Node interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

// NewNode returns a new Full or Light Node based on the config
// This is the entry point for composing a node, when compiling a node, you need to provide an executor
// Example executors can be found: TODO: add link
func NewNode(
	ctx context.Context,
	conf config.NodeConfig,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	p2pKey crypto.PrivKey,
	signingKey crypto.PrivKey,
	genesis *types.GenesisDoc,
	metricsProvider MetricsProvider,
	logger log.Logger,
) (Node, error) {
	if conf.Light {
		return newLightNode(
			conf,
			p2pKey,
			genesis,
			metricsProvider,
			logger,
		)
	}

	return newFullNode(
		conf,
		p2pKey,
		signingKey,
		genesis,
		exec,
		sequencer,
		metricsProvider,
		logger,
	)
}
