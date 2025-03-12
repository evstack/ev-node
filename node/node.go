package node

import (
	"context"

	"cosmossdk.io/log"
	cmtypes "github.com/cometbft/cometbft/types"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/rollkit/rollkit/config"
	coreda "github.com/rollkit/rollkit/core/da"
	coreexecutor "github.com/rollkit/rollkit/core/execution"
	coresequencer "github.com/rollkit/rollkit/core/sequencer"
	"github.com/rollkit/rollkit/pkg/service"
)

// Node is the interface for a rollup node
type Node interface {
	service.Service
}

// NewNode returns a new Full or Light Node based on the config
// This is the entry point for composing a node, when compiling a node, you need to provide an executor
// Example executors can be found: TODO: add link
func NewNode(
	ctx context.Context,
	conf config.Config,
	exec coreexecutor.Executor,
	sequencer coresequencer.Sequencer,
	dac coreda.Client,
	p2pKey crypto.PrivKey,
	signingKey crypto.PrivKey,
	genesis *cmtypes.GenesisDoc,
	metricsProvider MetricsProvider,
	logger log.Logger,
) (Node, error) {
	if conf.Node.Light {
		return newLightNode(conf, p2pKey, genesis, metricsProvider, logger)
	}

	return newFullNode(
		ctx,
		conf,
		p2pKey,
		signingKey,
		genesis,
		exec,
		sequencer,
		dac,
		metricsProvider,
		logger,
	)
}
