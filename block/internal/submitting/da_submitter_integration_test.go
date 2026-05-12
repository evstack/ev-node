package submitting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/rpc/server"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/rs/zerolog"
)

func TestNewDASubmitterSetsVisualizerWhenEnabled(t *testing.T) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.RPC.EnableDAVisualization = true
	cfg.Node.Aggregator = true

	daClient := mocks.NewMockClient(t)
	NewDASubmitter(
		daClient,
		cfg,
		genesis.Genesis{},
		common.DefaultBlockOptions(),
		common.NopMetrics(),
		zerolog.Nop(),
	)

	require.NotNil(t, server.GetDAVisualizationServer())
}
