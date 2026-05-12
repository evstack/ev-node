package submitting

import (
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
)

const (
	testHeaderNamespace = "test-headers"
	testDataNamespace   = "test-data"
)

func setupDASubmitterTest(t *testing.T) (*DASubmitter, store.Store, cache.Manager, *mocks.MockClient, genesis.Genesis) {
	t.Helper()

	ds := sync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = testHeaderNamespace
	cfg.DA.DataNamespace = testDataNamespace

	mockDA := mocks.NewMockClient(t)

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now(),
		ProposerAddress: []byte("test-proposer"),
	}

	daSubmitter := NewDASubmitter(
		mockDA,
		cfg,
		gen,
		common.DefaultBlockOptions(),
		common.NopMetrics(),
		zerolog.Nop(),
	)

	return daSubmitter, st, cm, mockDA, gen
}

func TestDASubmitter_NewDASubmitter(t *testing.T) {
	submitter, _, _, _, _ := setupDASubmitterTest(t)

	assert.NotNil(t, submitter)
	assert.NotNil(t, submitter.client)
	assert.NotNil(t, submitter.config)
	assert.NotNil(t, submitter.genesis)
}
