package block

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	cmcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	cmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cometbft/cometbft/proxy"
	cmtypes "github.com/cometbft/cometbft/types"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	goDA "github.com/rollkit/go-da"
	goDAMock "github.com/rollkit/go-da/mocks"
	goDATest "github.com/rollkit/go-da/test"
	"github.com/rollkit/go-sequencing"

	seqGRPC "github.com/rollkit/go-sequencing/proxy/grpc"
	"github.com/rollkit/rollkit/config"
	"github.com/rollkit/rollkit/da"
	"github.com/rollkit/rollkit/mempool"
	"github.com/rollkit/rollkit/state"
	"github.com/rollkit/rollkit/store"
	test "github.com/rollkit/rollkit/test/log"
	"github.com/rollkit/rollkit/test/mocks"
	"github.com/rollkit/rollkit/types"
)

// MockSequencerAddress is a sample address used by the mock sequencer
const MockSequencerAddress = "localhost:50051"

// WithinDuration asserts that the two durations are within the specified tolerance of each other.
func WithinDuration(t *testing.T, expected, actual, tolerance time.Duration) bool {
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff <= tolerance {
		return true
	}
	return assert.Fail(t, fmt.Sprintf("Not within duration.\nExpected: %v\nActual: %v\nTolerance: %v", expected, actual, tolerance))
}

// Returns a minimalistic block manager
func getManager(t *testing.T, backend goDA.DA) *Manager {
	logger := test.NewLogger(t)
	return &Manager{
		dalc:        da.NewDAClient(backend, -1, -1, nil, nil, logger),
		headerCache: NewHeaderCache(),
		logger:      logger,
	}
}

// getBlockBiggerThan generates a block with the given height bigger than the specified limit.
// func getBlockBiggerThan(blockHeight, limit uint64) (*types.SignedHeader, *types.Data, error) {
// 	for numTxs := 0; ; numTxs += 100 {
// 		header, data := types.GetRandomBlock(blockHeight, numTxs)
// 		blob, err := header.MarshalBinary()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		if uint64(len(blob)) > limit {
// 			return header, data, nil
// 		}
// 	}
// }

func TestInitialStateClean(t *testing.T) {
	const chainID = "TestInitialStateClean"
	require := require.New(t)
	genesisDoc, _ := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, chainID)
	genesis := &cmtypes.GenesisDoc{
		ChainID:       chainID,
		InitialHeight: 1,
		Validators:    genesisDoc.Validators,
		AppHash:       []byte("app hash"),
	}
	es, _ := store.NewDefaultInMemoryKVStore()
	emptyStore := store.New(es)
	s, err := getInitialState(emptyStore, genesis)
	require.NoError(err)
	require.Equal(s.LastBlockHeight, uint64(genesis.InitialHeight-1))
	require.Equal(uint64(genesis.InitialHeight), s.InitialHeight)
}

func TestInitialStateStored(t *testing.T) {
	chainID := "TestInitialStateStored"
	require := require.New(t)
	genesisDoc, _ := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, chainID)
	valset := types.GetRandomValidatorSet()
	genesis := &cmtypes.GenesisDoc{
		ChainID:       chainID,
		InitialHeight: 1,
		Validators:    genesisDoc.Validators,
		AppHash:       []byte("app hash"),
	}
	sampleState := types.State{
		ChainID:         chainID,
		InitialHeight:   1,
		LastBlockHeight: 100,
		Validators:      valset,
		NextValidators:  valset,
		LastValidators:  valset,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	es, _ := store.NewDefaultInMemoryKVStore()
	store := store.New(es)
	err := store.UpdateState(ctx, sampleState)
	require.NoError(err)
	s, err := getInitialState(store, genesis)
	require.NoError(err)
	require.Equal(s.LastBlockHeight, uint64(100))
	require.Equal(s.InitialHeight, uint64(1))
}

func TestHandleEmptyDataHash(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	// Mock store and data cache
	store := mocks.NewStore(t)
	dataCache := NewDataCache()

	// Setup the manager with the mock and data cache
	m := &Manager{
		store:     store,
		dataCache: dataCache,
	}

	// Define the test data
	headerHeight := 2
	header := &types.Header{
		DataHash: dataHashForEmptyTxs,
		BaseHeader: types.BaseHeader{
			Height: 2,
			Time:   uint64(time.Now().UnixNano()),
		},
	}

	// Mock data for the previous block
	lastData := &types.Data{}
	lastDataHash := lastData.Hash()

	// header.DataHash equals dataHashForEmptyTxs and no error occurs
	store.On("GetBlockData", ctx, uint64(headerHeight-1)).Return(nil, lastData, nil)

	// Execute the method under test
	m.handleEmptyDataHash(ctx, header)

	// Assertions
	store.AssertExpectations(t)

	// make sure that the store has the correct data
	d := dataCache.getData(header.Height())
	require.NotNil(d)
	require.Equal(d.Metadata.LastDataHash, lastDataHash)
	require.Equal(d.Metadata.ChainID, header.ChainID())
	require.Equal(d.Metadata.Height, header.Height())
	require.Equal(d.Metadata.Time, header.BaseHeader.Time)
}

func TestInitialStateUnexpectedHigherGenesis(t *testing.T) {
	require := require.New(t)
	genesisDoc, _ := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "TestInitialStateUnexpectedHigherGenesis")
	valset := types.GetRandomValidatorSet()
	genesis := &cmtypes.GenesisDoc{
		ChainID:       "TestInitialStateUnexpectedHigherGenesis",
		InitialHeight: 2,
		Validators:    genesisDoc.Validators,
		AppHash:       []byte("app hash"),
	}
	sampleState := types.State{
		ChainID:         "TestInitialStateUnexpectedHigherGenesis",
		InitialHeight:   1,
		LastBlockHeight: 0,
		Validators:      valset,
		NextValidators:  valset,
		LastValidators:  valset,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	es, _ := store.NewDefaultInMemoryKVStore()
	store := store.New(es)
	err := store.UpdateState(ctx, sampleState)
	require.NoError(err)
	_, err = getInitialState(store, genesis)
	require.EqualError(err, "genesis.InitialHeight (2) is greater than last stored state's LastBlockHeight (0)")
}

func TestSignVerifySignature(t *testing.T) {
	require := require.New(t)
	m := getManager(t, goDATest.NewDummyDA())
	payload := []byte("test")
	cases := []struct {
		name  string
		input cmcrypto.PrivKey
	}{
		{"ed25519", ed25519.GenPrivKey()},
		{"secp256k1", secp256k1.GenPrivKey()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pubKey := c.input.PubKey()
			signingKey, err := types.PrivKeyToSigningKey(c.input)
			require.NoError(err)
			m.proposerKey = signingKey
			signature, err := m.sign(payload)
			require.NoError(err)
			ok := pubKey.VerifySignature(payload, signature)
			require.True(ok)
		})
	}
}

func TestIsDAIncluded(t *testing.T) {
	require := require.New(t)

	// Create a minimalistic block manager
	m := &Manager{
		headerCache: NewHeaderCache(),
	}
	hash := types.Hash([]byte("hash"))

	// IsDAIncluded should return false for unseen hash
	require.False(m.IsDAIncluded(hash))

	// Set the hash as DAIncluded and verify IsDAIncluded returns true
	m.headerCache.setDAIncluded(hash.String())
	require.True(m.IsDAIncluded(hash))
}

func TestSubmitBlocksToMockDA(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name              string
		gasPrice          float64
		gasMultiplier     float64
		expectedGasPrices []float64
		isErrExpected     bool
	}{
		{"defaults", -1, -1, []float64{
			-1, -1, -1,
		}, false},
		{"fixed_gas_price", 1.0, -1, []float64{
			1.0, 1.0, 1.0,
		}, false},
		{"default_gas_price_with_multiplier", -1, 1.2, []float64{
			-1, -1, -1,
		}, false},
		{"fixed_gas_price_with_multiplier", 1.0, 1.2, []float64{
			1.0, 1.2, 1.2 * 1.2,
		}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDA := &goDAMock.MockDA{}
			m := getManager(t, mockDA)
			m.conf.DABlockTime = time.Millisecond
			m.conf.DAMempoolTTL = 1
			kvStore, err := store.NewDefaultInMemoryKVStore()
			require.NoError(t, err)
			m.store = store.New(kvStore)

			var blobs [][]byte
			header, data := types.GetRandomBlock(1, 5, "TestSubmitBlocksToMockDA")
			blob, err := header.MarshalBinary()
			require.NoError(t, err)

			err = m.store.SaveBlockData(ctx, header, data, &types.Signature{})
			require.NoError(t, err)
			m.store.SetHeight(ctx, 1)

			m.dalc.GasPrice = tc.gasPrice
			m.dalc.GasMultiplier = tc.gasMultiplier

			blobs = append(blobs, blob)
			// Set up the mock to
			// * throw timeout waiting for tx to be included exactly twice
			// * wait for tx to drop from mempool exactly DABlockTime * DAMempoolTTL seconds
			// * retry with a higher gas price
			// * successfully submit
			mockDA.On("MaxBlobSize", mock.Anything).Return(uint64(12345), nil)
			mockDA.
				On("Submit", mock.Anything, blobs, tc.expectedGasPrices[0], []byte(nil)).
				Return([][]byte{}, &goDA.ErrTxTimedOut{}).Once()
			mockDA.
				On("Submit", mock.Anything, blobs, tc.expectedGasPrices[1], []byte(nil)).
				Return([][]byte{}, &goDA.ErrTxTimedOut{}).Once()
			mockDA.
				On("Submit", mock.Anything, blobs, tc.expectedGasPrices[2], []byte(nil)).
				Return([][]byte{bytes.Repeat([]byte{0x00}, 8)}, nil)

			m.pendingHeaders, err = NewPendingHeaders(m.store, m.logger)
			require.NoError(t, err)
			err = m.submitHeadersToDA(ctx)
			require.NoError(t, err)
			mockDA.AssertExpectations(t)
		})
	}
}

// func TestSubmitBlocksToDA(t *testing.T) {
// 	assert := assert.New(t)
// 	require := require.New(t)
// 	ctx := context.Background()

// 	m := getManager(t, goDATest.NewDummyDA())

// 	maxDABlobSizeLimit, err := m.dalc.DA.MaxBlobSize(ctx)
// 	require.NoError(err)

// 	h1, d1 := func() ([]*types.SignedHeader, []*types.Data) {
// 		numBlocks, numTxs := 3, 5
// 		headers := make([]*types.SignedHeader, numBlocks)
// 		data := make([]*types.Data, numBlocks)
// 		headers[0], data[0] = types.GetRandomBlock(uint64(1), numTxs)
// 		headers[1], data[1], err = getBlockBiggerThan(2, maxDABlobSizeLimit)
// 		require.NoError(err)
// 		headers[2], data[2] = types.GetRandomBlock(uint64(3), numTxs)
// 		return headers, data
// 	}()
// 	h2, d2 := func() ([]*types.SignedHeader, []*types.Data) {
// 		numBlocks, numTxs := 3, 5
// 		headers := make([]*types.SignedHeader, numBlocks)
// 		data := make([]*types.Data, numBlocks)
// 		for i := 0; i < numBlocks-1; i++ {
// 			headers[i], data[i] = types.GetRandomBlock(uint64(i+1), numTxs)
// 		}
// 		headers[2], data[2], err = getBlockBiggerThan(3, maxDABlobSizeLimit)
// 		require.NoError(err)
// 		return headers, data
// 	}()
// 	h3, d3 := func() ([]*types.SignedHeader, []*types.Data) {
// 		// Find three blocks where two of them are under blob size limit
// 		// but adding the third one exceeds the blob size limit
// 		header1, data1 := types.GetRandomBlock(1, 100)
// 		blob1, err := header1.MarshalBinary()
// 		require.NoError(err)

// 		header2, data2 := types.GetRandomBlock(2, 100)
// 		blob2, err := header2.MarshalBinary()
// 		require.NoError(err)

// 		header3, data3, err := getBlockBiggerThan(3, maxDABlobSizeLimit-uint64(len(blob1)+len(blob2)))
// 		require.NoError(err)

// 		return []*types.SignedHeader{header1, header2, header3}, []*types.Data{data1, data2, data3}
// 	}()
// 	h4, d4 := types.GetRandomBlock(1, 5)
// 	h5, d5 := types.GetRandomBlock(2, 5)
// 	h6, d6 := types.GetRandomBlock(3, 5)
// 	testCases := []struct {
// 		name                        string
// 		headers                     []*types.SignedHeader
// 		data                        []*types.Data
// 		isErrExpected               bool
// 		expectedPendingBlocksLength int
// 		expectedDAIncludedHeight    uint64
// 	}{
// 		{
// 			name:                        "B is too big on its own. So A gets submitted but, B and C never get submitted",
// 			headers:                     h1,
// 			data:                        d1,
// 			isErrExpected:               true,
// 			expectedPendingBlocksLength: 2,
// 			expectedDAIncludedHeight:    1,
// 		},
// 		{
// 			name:                        "A and B are submitted successfully but C is too big on its own, so C never gets submitted",
// 			headers:                     h2,
// 			data:                        d2,
// 			isErrExpected:               true,
// 			expectedPendingBlocksLength: 1,
// 			expectedDAIncludedHeight:    2,
// 		},
// 		{
// 			name:                        "blocks A and B are submitted together without C because including C triggers blob size limit. C is submitted in a separate round",
// 			headers:                     h3,
// 			data:                        d3,
// 			isErrExpected:               false,
// 			expectedPendingBlocksLength: 0,
// 			expectedDAIncludedHeight:    3,
// 		},
// 		{
// 			name:                        "happy path, all blocks A, B, C combine to less than maxDABlobSize",
// 			headers:                     []*types.SignedHeader{h4, h5, h6},
// 			data:                        []*types.Data{d4, d5, d6},
// 			isErrExpected:               false,
// 			expectedPendingBlocksLength: 0,
// 			expectedDAIncludedHeight:    3,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		// there is a limitation of value size for underlying in-memory KV store, so (temporary) on-disk store is needed
// 		kvStore := getTempKVStore(t)
// 		m.store = store.New(kvStore)
// 		m.pendingHeaders, err = NewPendingHeaders(m.store, m.logger)
// 		require.NoError(err)
// 		t.Run(tc.name, func(t *testing.T) {
// 			// PendingBlocks depend on store, so blocks needs to be saved and height updated
// 			for i, header := range tc.headers {
// 				data := tc.data[i]
// 				require.NoError(m.store.SaveBlockData(ctx, header, data, &types.Signature{}))
// 			}
// 			m.store.SetHeight(ctx, uint64(len(tc.headers)))

// 			err := m.submitHeadersToDA(ctx)
// 			assert.Equal(tc.isErrExpected, err != nil)
// 			blocks, err := m.pendingHeaders.getPendingHeaders(ctx)
// 			assert.NoError(err)
// 			assert.Equal(tc.expectedPendingBlocksLength, len(blocks))

// 			// ensure that metadata is updated in KV store
// 			raw, err := m.store.GetMetadata(ctx, LastSubmittedHeightKey)
// 			require.NoError(err)
// 			lshInKV, err := strconv.ParseUint(string(raw), 10, 64)
// 			require.NoError(err)
// 			assert.Equal(m.store.Height(), lshInKV+uint64(tc.expectedPendingBlocksLength))

// 			// ensure that da included height is updated in KV store
// 			assert.Equal(tc.expectedDAIncludedHeight, m.GetDAIncludedHeight())
// 		})
// 	}
// }

// func getTempKVStore(t *testing.T) ds.TxnDatastore {
// 	dbPath, err := os.MkdirTemp("", t.Name())
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		_ = os.RemoveAll(dbPath)
// 	})
// 	kvStore, err := store.NewDefaultKVStore(os.TempDir(), dbPath, t.Name())
// 	require.NoError(t, err)
// 	return kvStore
// }

// Test_submitBlocksToDA_BlockMarshalErrorCase1: A itself has a marshalling error. So A, B and C never get submitted.
func Test_submitBlocksToDA_BlockMarshalErrorCase1(t *testing.T) {
	chainID := "Test_submitBlocksToDA_BlockMarshalErrorCase1"
	assert := assert.New(t)
	require := require.New(t)
	ctx := context.Background()

	m := getManager(t, goDATest.NewDummyDA())

	header1, data1 := types.GetRandomBlock(uint64(1), 5, chainID)
	header2, data2 := types.GetRandomBlock(uint64(2), 5, chainID)
	header3, data3 := types.GetRandomBlock(uint64(3), 5, chainID)

	store := mocks.NewStore(t)
	invalidateBlockHeader(header1)
	store.On("GetMetadata", ctx, LastSubmittedHeightKey).Return(nil, ds.ErrNotFound)
	store.On("GetBlockData", ctx, uint64(1)).Return(header1, data1, nil)
	store.On("GetBlockData", ctx, uint64(2)).Return(header2, data2, nil)
	store.On("GetBlockData", ctx, uint64(3)).Return(header3, data3, nil)
	store.On("Height").Return(uint64(3))

	m.store = store

	var err error
	m.pendingHeaders, err = NewPendingHeaders(store, m.logger)
	require.NoError(err)

	err = m.submitHeadersToDA(ctx)
	assert.ErrorContains(err, "failed to submit all blocks to DA layer")
	blocks, err := m.pendingHeaders.getPendingHeaders(ctx)
	assert.NoError(err)
	assert.Equal(3, len(blocks))
}

// Test_submitBlocksToDA_BlockMarshalErrorCase2: A and B are fair blocks, but C has a marshalling error
// - Block A and B get submitted to DA layer not block C
func Test_submitBlocksToDA_BlockMarshalErrorCase2(t *testing.T) {
	chainID := "Test_submitBlocksToDA_BlockMarshalErrorCase2"
	assert := assert.New(t)
	require := require.New(t)
	ctx := context.Background()

	m := getManager(t, goDATest.NewDummyDA())

	header1, data1 := types.GetRandomBlock(uint64(1), 5, chainID)
	header2, data2 := types.GetRandomBlock(uint64(2), 5, chainID)
	header3, data3 := types.GetRandomBlock(uint64(3), 5, chainID)

	store := mocks.NewStore(t)
	invalidateBlockHeader(header3)
	store.On("SetMetadata", ctx, DAIncludedHeightKey, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}).Return(nil)
	store.On("SetMetadata", ctx, DAIncludedHeightKey, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}).Return(nil)
	store.On("SetMetadata", ctx, LastSubmittedHeightKey, []byte(strconv.FormatUint(2, 10))).Return(nil)
	store.On("GetMetadata", ctx, LastSubmittedHeightKey).Return(nil, ds.ErrNotFound)
	store.On("GetBlockData", ctx, uint64(1)).Return(header1, data1, nil)
	store.On("GetBlockData", ctx, uint64(2)).Return(header2, data2, nil)
	store.On("GetBlockData", ctx, uint64(3)).Return(header3, data3, nil)
	store.On("Height").Return(uint64(3))

	m.store = store

	var err error
	m.pendingHeaders, err = NewPendingHeaders(store, m.logger)
	require.NoError(err)
	err = m.submitHeadersToDA(ctx)
	assert.ErrorContains(err, "failed to submit all blocks to DA layer")
	blocks, err := m.pendingHeaders.getPendingHeaders(ctx)
	assert.NoError(err)
	assert.Equal(1, len(blocks))
}

// invalidateBlockHeader results in a block header that produces a marshalling error
func invalidateBlockHeader(header *types.SignedHeader) {
	for i := range header.Validators.Validators {
		header.Validators.Validators[i] = &cmtypes.Validator{
			Address:          []byte(""),
			PubKey:           nil,
			VotingPower:      -1,
			ProposerPriority: 0,
		}
	}
}

func Test_isProposer(t *testing.T) {
	require := require.New(t)

	type args struct {
		state         types.State
		signerPrivKey crypto.PrivKey
	}
	tests := []struct {
		name       string
		args       args
		isProposer bool
		err        error
	}{
		{
			name: "Signing key matches genesis proposer public key",
			args: func() args {
				genesisData, privKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "Test_isProposer")
				s, err := types.NewFromGenesisDoc(genesisData)
				require.NoError(err)
				signingKey, err := types.PrivKeyToSigningKey(privKey)
				require.NoError(err)
				return args{
					s,
					signingKey,
				}
			}(),
			isProposer: true,
			err:        nil,
		},
		{
			name: "Signing key does not match genesis proposer public key",
			args: func() args {
				genesisData, _ := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "Test_isProposer")
				s, err := types.NewFromGenesisDoc(genesisData)
				require.NoError(err)

				randomPrivKey := ed25519.GenPrivKey()
				signingKey, err := types.PrivKeyToSigningKey(randomPrivKey)
				require.NoError(err)
				return args{
					s,
					signingKey,
				}
			}(),
			isProposer: false,
			err:        nil,
		},
		{
			name: "No validators found in genesis",
			args: func() args {
				genesisData, privKey := types.GetGenesisWithPrivkey(types.DefaultSigningKeyType, "Test_isProposer")
				genesisData.Validators = nil
				s, err := types.NewFromGenesisDoc(genesisData)
				require.NoError(err)

				signingKey, err := types.PrivKeyToSigningKey(privKey)
				require.NoError(err)
				return args{
					s,
					signingKey,
				}
			}(),
			isProposer: false,
			err:        ErrNoValidatorsInState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isProposer, err := isProposer(tt.args.signerPrivKey, tt.args.state)
			if !errors.Is(err, tt.err) {
				t.Errorf("isProposer() error = %v, expected err %v", err, tt.err)
				return
			}
			if isProposer != tt.isProposer {
				t.Errorf("isProposer() = %v, expected %v", isProposer, tt.isProposer)
			}
		})
	}
}

func Test_publishBlock_ManagerNotProposer(t *testing.T) {
	require := require.New(t)
	m := getManager(t, &goDAMock.MockDA{})
	m.isProposer = false
	err := m.publishBlock(context.Background())
	require.ErrorIs(err, ErrNotProposer)
}

func TestManager_publishBlock(t *testing.T) {
	mockStore := new(mocks.Store)
	mockLogger := new(test.MockLogger)
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	var mockAppHash []byte
	_, err := rand.Read(mockAppHash[:])
	require.NoError(err)

	app := &mocks.Application{}
	app.On("CheckTx", mock.Anything, mock.Anything).Return(&abci.ResponseCheckTx{}, nil)
	app.On("Commit", mock.Anything, mock.Anything).Return(&abci.ResponseCommit{}, nil)
	app.On("PrepareProposal", mock.Anything, mock.Anything).Return(func(_ context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		return &abci.ResponsePrepareProposal{
			Txs: req.Txs,
		}, nil
	})
	app.On("ProcessProposal", mock.Anything, mock.Anything).Return(&abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil)
	app.On("FinalizeBlock", mock.Anything, mock.Anything).Return(
		func(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
			txResults := make([]*abci.ExecTxResult, len(req.Txs))
			for idx := range req.Txs {
				txResults[idx] = &abci.ExecTxResult{
					Code: abci.CodeTypeOK,
				}
			}

			return &abci.ResponseFinalizeBlock{
				TxResults: txResults,
				AppHash:   mockAppHash,
			}, nil
		},
	)

	client, err := proxy.NewLocalClientCreator(app).NewABCIClient()
	require.NoError(err)
	require.NotNil(client)

	vKey := ed25519.GenPrivKey()
	validators := []*cmtypes.Validator{
		{
			Address:          vKey.PubKey().Address(),
			PubKey:           vKey.PubKey(),
			VotingPower:      int64(100),
			ProposerPriority: int64(1),
		},
	}

	lastState := types.State{}
	lastState.ConsensusParams.Block = &cmproto.BlockParams{}
	lastState.ConsensusParams.Block.MaxBytes = 100
	lastState.ConsensusParams.Block.MaxGas = 100000
	lastState.ConsensusParams.Abci = &cmproto.ABCIParams{VoteExtensionsEnableHeight: 0}
	lastState.Validators = cmtypes.NewValidatorSet(validators)
	lastState.NextValidators = cmtypes.NewValidatorSet(validators)
	lastState.LastValidators = cmtypes.NewValidatorSet(validators)

	chainID := "TestManager_publishBlock"
	mpool := mempool.NewCListMempool(cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(client, proxy.NopMetrics()), 0)
	seqClient := seqGRPC.NewClient()
	require.NoError(seqClient.Start(
		MockSequencerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	mpoolReaper := mempool.NewCListMempoolReaper(mpool, []byte(chainID), seqClient, logger)
	executor := state.NewBlockExecutor(vKey.PubKey().Address(), chainID, mpool, mpoolReaper, proxy.NewAppConnConsensus(client, proxy.NopMetrics()), nil, 100, logger, state.NopMetrics())

	signingKey, err := types.PrivKeyToSigningKey(vKey)
	require.NoError(err)
	m := &Manager{
		lastState:    lastState,
		lastStateMtx: new(sync.RWMutex),
		headerCache:  NewHeaderCache(),
		dataCache:    NewDataCache(),
		executor:     executor,
		store:        mockStore,
		logger:       mockLogger,
		genesis: &cmtypes.GenesisDoc{
			ChainID:       chainID,
			InitialHeight: 1,
			AppHash:       []byte("app hash"),
		},
		conf: config.BlockManagerConfig{
			BlockTime:      time.Second,
			LazyAggregator: false,
		},
		isProposer:  true,
		proposerKey: signingKey,
		metrics:     NopMetrics(),
	}

	t.Run("height should not be updated if saving block responses fails", func(t *testing.T) {
		mockStore.On("Height").Return(uint64(0))
		signature := types.Signature([]byte{1, 1, 1})
		header, data, err := executor.CreateBlock(0, &signature, abci.ExtendedCommitInfo{}, []byte{}, lastState, cmtypes.Txs{}, time.Now())
		require.NoError(err)
		require.NotNil(header)
		require.NotNil(data)
		assert.Equal(uint64(0), header.Height())
		dataHash := data.Hash()
		header.DataHash = dataHash

		// Update the signature on the block to current from last
		voteBytes := header.Header.MakeCometBFTVote()
		signature, _ = vKey.Sign(voteBytes)
		header.Signature = signature
		header.Validators = lastState.Validators

		mockStore.On("GetBlockData", mock.Anything, uint64(1)).Return(header, data, nil).Once()
		mockStore.On("SaveBlockData", mock.Anything, header, data, mock.Anything).Return(nil).Once()
		mockStore.On("SaveBlockResponses", mock.Anything, uint64(0), mock.Anything).Return(SaveBlockResponsesError{}).Once()

		ctx := context.Background()
		err = m.publishBlock(ctx)
		assert.ErrorAs(err, &SaveBlockResponsesError{})

		mockStore.AssertExpectations(t)
	})
}

func TestManager_getRemainingSleep(t *testing.T) {
	tests := []struct {
		name          string
		manager       *Manager
		start         time.Time
		expectedSleep time.Duration
	}{
		{
			name: "Normal aggregation, elapsed < interval",
			manager: &Manager{
				conf: config.BlockManagerConfig{
					BlockTime:      10 * time.Second,
					LazyBlockTime:  20 * time.Second,
					LazyAggregator: false,
				},
			},
			start:         time.Now().Add(-5 * time.Second),
			expectedSleep: 5 * time.Second,
		},
		{
			name: "Normal aggregation, elapsed >= interval",
			manager: &Manager{
				conf: config.BlockManagerConfig{
					BlockTime:      10 * time.Second,
					LazyBlockTime:  20 * time.Second,
					LazyAggregator: false,
				},
			},
			start:         time.Now().Add(-15 * time.Second),
			expectedSleep: 0,
		},
		{
			name: "Lazy aggregation, building block, elapsed < interval",
			manager: &Manager{
				conf: config.BlockManagerConfig{
					BlockTime:      10 * time.Second,
					LazyBlockTime:  20 * time.Second,
					LazyAggregator: true,
				},
				buildingBlock: true,
			},
			start:         time.Now().Add(-5 * time.Second),
			expectedSleep: 5 * time.Second,
		},
		{
			name: "Lazy aggregation, building block, elapsed >= interval",
			manager: &Manager{
				conf: config.BlockManagerConfig{
					BlockTime:      10 * time.Second,
					LazyBlockTime:  20 * time.Second,
					LazyAggregator: true,
				},
				buildingBlock: true,
			},
			start:         time.Now().Add(-15 * time.Second),
			expectedSleep: (10 * time.Second * time.Duration(defaultLazySleepPercent) / 100),
		},
		{
			name: "Lazy aggregation, not building block, elapsed < interval",
			manager: &Manager{
				conf: config.BlockManagerConfig{
					BlockTime:      10 * time.Second,
					LazyBlockTime:  20 * time.Second,
					LazyAggregator: true,
				},
				buildingBlock: false,
			},
			start:         time.Now().Add(-5 * time.Second),
			expectedSleep: 15 * time.Second,
		},
		{
			name: "Lazy aggregation, not building block, elapsed >= interval",
			manager: &Manager{
				conf: config.BlockManagerConfig{
					BlockTime:      10 * time.Second,
					LazyBlockTime:  20 * time.Second,
					LazyAggregator: true,
				},
				buildingBlock: false,
			},
			start:         time.Now().Add(-25 * time.Second),
			expectedSleep: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSleep := tt.manager.getRemainingSleep(tt.start)
			// Allow for a small difference, e.g., 5 millisecond
			assert.True(t, WithinDuration(t, tt.expectedSleep, actualSleep, 5*time.Millisecond))
		})
	}
}

// TestAggregationLoop tests the AggregationLoop function
func TestAggregationLoop(t *testing.T) {
	mockStore := new(mocks.Store)
	mockLogger := new(test.MockLogger)

	m := &Manager{
		store:  mockStore,
		logger: mockLogger,
		genesis: &cmtypes.GenesisDoc{
			ChainID:       "myChain",
			InitialHeight: 1,
			AppHash:       []byte("app hash"),
		},
		conf: config.BlockManagerConfig{
			BlockTime:      time.Second,
			LazyAggregator: false,
		},
		bq: NewBatchQueue(),
	}

	mockStore.On("Height").Return(uint64(0))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go m.AggregationLoop(ctx)

	// Wait for the function to complete or timeout
	<-ctx.Done()

	mockStore.AssertExpectations(t)
}

// TestLazyAggregationLoop tests the lazyAggregationLoop function
func TestLazyAggregationLoop(t *testing.T) {
	mockLogger := new(test.MockLogger)

	m := &Manager{
		logger: mockLogger,
		conf: config.BlockManagerConfig{
			BlockTime:      time.Second,
			LazyAggregator: true,
		},
		bq: NewBatchQueue(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	blockTimer := time.NewTimer(m.conf.BlockTime)
	defer blockTimer.Stop()

	go m.lazyAggregationLoop(ctx, blockTimer)
	m.bq.notifyCh <- struct{}{}

	// Wait for the function to complete or timeout
	<-ctx.Done()
}

// TestNormalAggregationLoop tests the normalAggregationLoop function
func TestNormalAggregationLoop(t *testing.T) {
	mockLogger := new(test.MockLogger)

	m := &Manager{
		logger: mockLogger,
		conf: config.BlockManagerConfig{
			BlockTime: time.Second,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	blockTimer := time.NewTimer(m.conf.BlockTime)
	defer blockTimer.Stop()

	go m.normalAggregationLoop(ctx, blockTimer)

	// Wait for the function to complete or timeout
	<-ctx.Done()
}

func TestGetTxsFromBatch_NoBatch(t *testing.T) {
	// Mocking a manager with an empty batch queue
	m := &Manager{
		bq: &BatchQueue{queue: nil}, // No batch available
	}

	// Call the method and assert the results
	txs, timestamp, err := m.getTxsFromBatch()

	// Assertions
	assert.Nil(t, txs, "Transactions should be nil when no batch exists")
	assert.Nil(t, timestamp, "Timestamp should be nil when no batch exists")
	assert.Equal(t, ErrNoBatch, err, "Expected ErrNoBatch error")
}

func TestGetTxsFromBatch_EmptyBatch(t *testing.T) {
	// Mocking a manager with an empty batch
	m := &Manager{
		bq: &BatchQueue{queue: []BatchWithTime{
			{Batch: &sequencing.Batch{Transactions: nil}, Time: time.Now()},
		}},
	}

	// Call the method and assert the results
	txs, timestamp, err := m.getTxsFromBatch()

	// Assertions
	require.NoError(t, err, "Expected no error for empty batch")
	assert.Empty(t, txs, "Transactions should be empty when batch has no transactions")
	assert.NotNil(t, timestamp, "Timestamp should not be nil for empty batch")
}

func TestGetTxsFromBatch_ValidBatch(t *testing.T) {
	// Mocking a manager with a valid batch
	m := &Manager{
		bq: &BatchQueue{queue: []BatchWithTime{
			{Batch: &sequencing.Batch{Transactions: [][]byte{[]byte("tx1"), []byte("tx2")}}, Time: time.Now()},
		}},
	}

	// Call the method and assert the results
	txs, timestamp, err := m.getTxsFromBatch()

	// Assertions
	require.NoError(t, err, "Expected no error for valid batch")
	assert.Len(t, txs, 2, "Expected 2 transactions")
	assert.NotNil(t, timestamp, "Timestamp should not be nil for valid batch")
	assert.Equal(t, cmtypes.Txs{cmtypes.Tx([]byte("tx1")), cmtypes.Tx([]byte("tx2"))}, txs, "Transactions do not match")
}
