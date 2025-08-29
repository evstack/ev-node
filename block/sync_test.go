package block

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goheaderstore "github.com/celestiaorg/go-header/store"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/cache"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// setupManagerForSyncLoopTest initializes a Manager instance suitable for SyncLoop testing.
func setupManagerForSyncLoopTest(t *testing.T, initialState types.State) (
	*Manager,
	*mocks.MockStore,
	*mocks.MockExecutor,
	context.Context,
	context.CancelFunc,
	chan daHeightEvent,
	*uint64,
) {
	t.Helper()

	mockStore := mocks.NewMockStore(t)
	mockExec := mocks.NewMockExecutor(t)

	heightInCh := make(chan daHeightEvent, 10)

	headerStoreCh := make(chan struct{}, 1)
	dataStoreCh := make(chan struct{}, 1)
	retrieveCh := make(chan struct{}, 1)

	cfg := config.DefaultConfig
	cfg.DA.BlockTime.Duration = 100 * time.Millisecond
	cfg.Node.BlockTime.Duration = 50 * time.Millisecond
	genesisDoc := &genesis.Genesis{ChainID: "syncLoopTest"}

	// Manager setup
	m := &Manager{
		store:                            mockStore,
		exec:                             mockExec,
		config:                           cfg,
		genesis:                          *genesisDoc,
		lastState:                        initialState,
		lastStateMtx:                     new(sync.RWMutex),
		logger:                           zerolog.Nop(),
		headerCache:                      cache.NewCache[types.SignedHeader](),
		dataCache:                        cache.NewCache[types.Data](),
		heightInCh:                       heightInCh,
		headerStoreCh:                    headerStoreCh,
		dataStoreCh:                      dataStoreCh,
		retrieveCh:                       retrieveCh,
		daHeight:                         &atomic.Uint64{},
		metrics:                          NopMetrics(),
		headerStore:                      &goheaderstore.Store[*types.SignedHeader]{},
		dataStore:                        &goheaderstore.Store[*types.Data]{},
		syncNodeSignaturePayloadProvider: types.DefaultSyncNodeSignatureBytesProvider,
		validatorHasherProvider:          types.DefaultValidatorHasherProvider,
	}
	m.daHeight.Store(initialState.DAHeight)

	ctx, cancel := context.WithCancel(context.Background())

	currentMockHeight := initialState.LastBlockHeight
	heightPtr := &currentMockHeight

	mockStore.On("Height", mock.Anything).Return(func(context.Context) uint64 {
		return *heightPtr
	}, nil).Maybe()

	return m, mockStore, mockExec, ctx, cancel, heightInCh, heightPtr
}

// TestSyncLoop_ProcessSingleBlock_HeaderFirst verifies that the sync loop processes a single block when the header arrives before the data.
// 1. Header for H+1 arrives.
// 2. Data for H+1 arrives.
// 3. Block H+1 is successfully validated, applied, and committed.
// 4. State is updated.
// 5. Caches are cleared.
func TestSyncLoop_ProcessSingleBlock_HeaderFirst(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	initialHeight := uint64(10)
	initialState := types.State{
		LastBlockHeight: initialHeight,
		AppHash:         []byte("initial_app_hash"),
		ChainID:         "syncLoopTest",
		DAHeight:        5,
	}
	newHeight := initialHeight + 1
	daHeight := initialState.DAHeight

	m, mockStore, mockExec, ctx, cancel, heightInCh, _ := setupManagerForSyncLoopTest(t, initialState)
	defer cancel()

	// Create test block data
	header, data, privKey := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: newHeight, NTxs: 2}, initialState.ChainID, initialState.AppHash)
	require.NotNil(header)
	require.NotNil(data)
	require.NotNil(privKey)

	expectedNewAppHash := []byte("new_app_hash")
	expectedNewState, err := initialState.NextState(header.Header, expectedNewAppHash)
	require.NoError(err)

	syncChan := make(chan struct{})
	var txs [][]byte
	for _, tx := range data.Txs {
		txs = append(txs, tx)
	}
	mockExec.On("ExecuteTxs", mock.Anything, txs, newHeight, header.Time(), initialState.AppHash).
		Return(expectedNewAppHash, uint64(100), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, header, data, &header.Signature).Return(nil).Once()

	mockStore.On("UpdateState", mock.Anything, expectedNewState).Return(nil).Run(func(args mock.Arguments) { close(syncChan) }).Once()

	mockStore.On("SetHeight", mock.Anything, newHeight).Return(nil).Once()

	ctx, loopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.SyncLoop(ctx, make(chan<- error))
	}()

	t.Logf("Sending height event for height %d", newHeight)
	heightInCh <- daHeightEvent{Header: header, Data: data, DaHeight: daHeight}

	t.Log("Waiting for sync to complete...")
	wg.Wait()

	select {
	case <-syncChan:
		t.Log("Sync completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for sync to complete")
	}

	mockStore.AssertExpectations(t)
	mockExec.AssertExpectations(t)

	finalState := m.GetLastState()
	assert.Equal(expectedNewState.LastBlockHeight, finalState.LastBlockHeight)
	assert.Equal(expectedNewState.AppHash, finalState.AppHash)
	assert.Equal(expectedNewState.LastBlockTime, finalState.LastBlockTime)
	assert.Equal(expectedNewState.DAHeight, finalState.DAHeight)

	// Assert caches are cleared for the processed height
	assert.Nil(m.headerCache.GetItem(newHeight), "Header cache should be cleared for processed height")
	assert.Nil(m.dataCache.GetItem(newHeight), "Data cache should be cleared for processed height")
}

// TestSyncLoop_ProcessSingleBlock_DataFirst verifies that the sync loop processes a single block when the data arrives before the header.
// 1. Data for H+1 arrives.
// 2. Header for H+1 arrives.
// 3. Block H+1 is successfully validated, applied, and committed.
// 4. State is updated.
// 5. Caches are cleared.
func TestSyncLoop_ProcessSingleBlock_DataFirst(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	initialHeight := uint64(20)
	initialState := types.State{
		LastBlockHeight: initialHeight,
		AppHash:         []byte("initial_app_hash_data_first"),
		ChainID:         "syncLoopTest",
		DAHeight:        15,
	}
	newHeight := initialHeight + 1
	daHeight := initialState.DAHeight

	m, mockStore, mockExec, ctx, cancel, heightInCh, _ := setupManagerForSyncLoopTest(t, initialState)
	defer cancel()

	// Create test block data
	header, data, privKey := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: newHeight, NTxs: 3}, initialState.ChainID, initialState.AppHash)
	require.NotNil(header)
	require.NotNil(data)
	require.NotNil(privKey)

	expectedNewAppHash := []byte("new_app_hash_data_first")
	expectedNewState, err := initialState.NextState(header.Header, expectedNewAppHash)
	require.NoError(err)

	syncChan := make(chan struct{})
	var txs [][]byte
	for _, tx := range data.Txs {
		txs = append(txs, tx)
	}

	mockExec.On("ExecuteTxs", mock.Anything, txs, newHeight, header.Time(), initialState.AppHash).
		Return(expectedNewAppHash, uint64(100), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, header, data, &header.Signature).Return(nil).Once()
	mockStore.On("UpdateState", mock.Anything, expectedNewState).Return(nil).Run(func(args mock.Arguments) { close(syncChan) }).Once()
	mockStore.On("SetHeight", mock.Anything, newHeight).Return(nil).Once()

	ctx, loopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.SyncLoop(ctx, make(chan<- error))
	}()

	t.Logf("Sending height event for height %d", newHeight)
	heightInCh <- daHeightEvent{Header: header, Data: data, DaHeight: daHeight}

	t.Log("Waiting for sync to complete...")

	wg.Wait()

	select {
	case <-syncChan:
		t.Log("Sync completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for sync to complete")
	}

	mockStore.AssertExpectations(t)
	mockExec.AssertExpectations(t)

	finalState := m.GetLastState()
	assert.Equal(expectedNewState.LastBlockHeight, finalState.LastBlockHeight)
	assert.Equal(expectedNewState.AppHash, finalState.AppHash)
	assert.Equal(expectedNewState.LastBlockTime, finalState.LastBlockTime)
	assert.Equal(expectedNewState.DAHeight, finalState.DAHeight)

	// Assert caches are cleared for the processed height
	assert.Nil(m.headerCache.GetItem(newHeight), "Header cache should be cleared for processed height")
	assert.Nil(m.dataCache.GetItem(newHeight), "Data cache should be cleared for processed height")
}

// TestSyncLoop_ProcessMultipleBlocks_Sequentially verifies that the sync loop processes multiple blocks arriving in order.
// 1. Events for H+1 arrive (header then data).
// 2. Block H+1 is processed.
// 3. Events for H+2 arrive (header then data).
// 4. Block H+2 is processed.
// 5. Final state is H+2.
func TestSyncLoop_ProcessMultipleBlocks_Sequentially(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	initialHeight := uint64(30)
	initialState := types.State{
		LastBlockHeight: initialHeight,
		AppHash:         []byte("initial_app_hash_multi"),
		ChainID:         "syncLoopTest",
		DAHeight:        25,
	}
	heightH1 := initialHeight + 1
	heightH2 := initialHeight + 2
	daHeight := initialState.DAHeight

	m, mockStore, mockExec, ctx, cancel, heightInCh, heightPtr := setupManagerForSyncLoopTest(t, initialState)
	defer cancel()

	// --- Block H+1 Data ---
	headerH1, dataH1, privKeyH1 := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: heightH1, NTxs: 1}, initialState.ChainID, initialState.AppHash)
	require.NotNil(headerH1)
	require.NotNil(dataH1)
	require.NotNil(privKeyH1)

	expectedNewAppHashH1 := []byte("app_hash_h1")
	expectedNewStateH1, err := initialState.NextState(headerH1.Header, expectedNewAppHashH1)
	require.NoError(err)

	var txsH1 [][]byte
	for _, tx := range dataH1.Txs {
		txsH1 = append(txsH1, tx)
	}

	// --- Block H+2 Data ---
	headerH2, dataH2, privKeyH2 := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: heightH2, NTxs: 2}, initialState.ChainID, expectedNewAppHashH1)
	require.NotNil(headerH2)
	require.NotNil(dataH2)
	require.NotNil(privKeyH2)

	expectedNewAppHashH2 := []byte("app_hash_h2")
	expectedNewStateH2, err := expectedNewStateH1.NextState(headerH2.Header, expectedNewAppHashH2)
	require.NoError(err)

	var txsH2 [][]byte
	for _, tx := range dataH2.Txs {
		txsH2 = append(txsH2, tx)
	}

	syncChanH1 := make(chan struct{})
	syncChanH2 := make(chan struct{})

	// --- Mock Expectations for H+1 ---

	mockExec.On("ExecuteTxs", mock.Anything, txsH1, heightH1, headerH1.Time(), initialState.AppHash).
		Return(expectedNewAppHashH1, uint64(100), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, headerH1, dataH1, &headerH1.Signature).Return(nil).Once()
	mockStore.On("UpdateState", mock.Anything, expectedNewStateH1).Return(nil).Run(func(args mock.Arguments) { close(syncChanH1) }).Once()
	mockStore.On("SetHeight", mock.Anything, heightH1).Return(nil).
		Run(func(args mock.Arguments) {
			newHeight := args.Get(1).(uint64)
			*heightPtr = newHeight // Update the mocked height
			t.Logf("Mock SetHeight called for H+1, updated mock height to %d", newHeight)
		}).
		Once()

	// --- Mock Expectations for H+2 ---
	mockExec.On("ExecuteTxs", mock.Anything, txsH2, heightH2, headerH2.Time(), expectedNewAppHashH1).
		Return(expectedNewAppHashH2, uint64(100), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, headerH2, dataH2, &headerH2.Signature).Return(nil).Once()
	mockStore.On("UpdateState", mock.Anything, expectedNewStateH2).Return(nil).Run(func(args mock.Arguments) { close(syncChanH2) }).Once()
	mockStore.On("SetHeight", mock.Anything, heightH2).Return(nil).
		Run(func(args mock.Arguments) {
			newHeight := args.Get(1).(uint64)
			*heightPtr = newHeight // Update the mocked height
			t.Logf("Mock SetHeight called for H+2, updated mock height to %d", newHeight)
		}).
		Once()

	ctx, loopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.SyncLoop(ctx, make(chan<- error))
		t.Log("SyncLoop exited.")
	}()

	// --- Process H+1 ---
	heightInCh <- daHeightEvent{Header: headerH1, Data: dataH1, DaHeight: daHeight}
	t.Log("Waiting for Sync H+1 to complete...")

	select {
	case <-syncChanH1:
		t.Log("Sync H+1 completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for sync H+1 to complete")
	}

	// --- Process H+2 ---
	heightInCh <- daHeightEvent{Header: headerH2, Data: dataH2, DaHeight: daHeight}

	select {
	case <-syncChanH2:
		t.Log("Sync H+2 completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for sync H+2 to complete")
	}

	t.Log("Waiting for SyncLoop H+2 to complete...")

	wg.Wait()

	mockStore.AssertExpectations(t)
	mockExec.AssertExpectations(t)

	finalState := m.GetLastState()
	assert.Equal(expectedNewStateH2.LastBlockHeight, finalState.LastBlockHeight)
	assert.Equal(expectedNewStateH2.AppHash, finalState.AppHash)
	assert.Equal(expectedNewStateH2.LastBlockTime, finalState.LastBlockTime)
	assert.Equal(expectedNewStateH2.DAHeight, finalState.DAHeight)

	assert.Nil(m.headerCache.GetItem(heightH1), "Header cache should be cleared for H+1")
	assert.Nil(m.dataCache.GetItem(heightH1), "Data cache should be cleared for H+1")
	assert.Nil(m.headerCache.GetItem(heightH2), "Header cache should be cleared for H+2")
	assert.Nil(m.dataCache.GetItem(heightH2), "Data cache should be cleared for H+2")
}

// TestSyncLoop_ProcessBlocks_OutOfOrderArrival verifies that the sync loop can handle blocks arriving out of order.
// 1. Events for H+2 arrive (header then data). Block H+2 is cached.
// 2. Events for H+1 arrive (header then data).
// 3. Block H+1 is processed.
// 4. Block H+2 is processed immediately after H+1 from the cache.
// 5. Final state is H+2.
func TestSyncLoop_ProcessBlocks_OutOfOrderArrival(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	initialHeight := uint64(40)
	initialState := types.State{
		LastBlockHeight: initialHeight,
		AppHash:         []byte("initial_app_hash_ooo"),
		ChainID:         "syncLoopTest",
		DAHeight:        35,
	}
	heightH1 := initialHeight + 1
	heightH2 := initialHeight + 2
	daHeight := initialState.DAHeight

	m, mockStore, mockExec, ctx, cancel, heightInCh, heightPtr := setupManagerForSyncLoopTest(t, initialState)
	defer cancel()

	// --- Block H+1 Data ---
	headerH1, dataH1, privKeyH1 := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: heightH1, NTxs: 1}, initialState.ChainID, initialState.AppHash)
	require.NotNil(headerH1)
	require.NotNil(dataH1)
	require.NotNil(privKeyH1)

	appHashH1 := []byte("app_hash_h1_ooo")
	expectedNewStateH1, err := initialState.NextState(headerH1.Header, appHashH1)
	require.NoError(err)

	var txsH1 [][]byte
	for _, tx := range dataH1.Txs {
		txsH1 = append(txsH1, tx)
	}

	// --- Block H+2 Data ---
	headerH2, dataH2, privKeyH2 := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: heightH2, NTxs: 2}, initialState.ChainID, appHashH1)
	require.NotNil(headerH2)
	require.NotNil(dataH2)
	require.NotNil(privKeyH2)

	appHashH2 := []byte("app_hash_h2_ooo")
	expectedStateH2, err := expectedNewStateH1.NextState(headerH2.Header, appHashH2)
	require.NoError(err)

	var txsH2 [][]byte
	for _, tx := range dataH2.Txs {
		txsH2 = append(txsH2, tx)
	}

	syncChanH1 := make(chan struct{})
	syncChanH2 := make(chan struct{})

	// --- Mock Expectations for H+1 (will be called first despite arrival order) ---
	mockStore.On("Height", mock.Anything).Return(initialHeight, nil).Maybe()
	mockExec.On("Validate", mock.Anything, &headerH1.Header, dataH1).Return(nil).Maybe()
	mockExec.On("ExecuteTxs", mock.Anything, txsH1, heightH1, headerH1.Time(), initialState.AppHash).
		Return(appHashH1, uint64(100), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, headerH1, dataH1, &headerH1.Signature).Return(nil).Once()
	mockStore.On("UpdateState", mock.Anything, expectedNewStateH1).Return(nil).
		Run(func(args mock.Arguments) { close(syncChanH1) }).
		Once()
	mockStore.On("SetHeight", mock.Anything, heightH1).Return(nil).
		Run(func(args mock.Arguments) {
			newHeight := args.Get(1).(uint64)
			*heightPtr = newHeight // Update the mocked height
			t.Logf("Mock SetHeight called for H+2, updated mock height to %d", newHeight)
		}).
		Once()

	// --- Mock Expectations for H+2 (will be called second) ---
	mockStore.On("Height", mock.Anything).Return(heightH1, nil).Maybe()
	mockExec.On("Validate", mock.Anything, &headerH2.Header, dataH2).Return(nil).Maybe()
	mockExec.On("ExecuteTxs", mock.Anything, txsH2, heightH2, headerH2.Time(), expectedNewStateH1.AppHash).
		Return(appHashH2, uint64(1), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, headerH2, dataH2, &headerH2.Signature).Return(nil).Once()
	mockStore.On("SetHeight", mock.Anything, heightH2).Return(nil).
		Run(func(args mock.Arguments) {
			newHeight := args.Get(1).(uint64)
			*heightPtr = newHeight // Update the mocked height
			t.Logf("Mock SetHeight called for H+2, updated mock height to %d", newHeight)
		}).
		Once()
	mockStore.On("UpdateState", mock.Anything, expectedStateH2).Return(nil).
		Run(func(args mock.Arguments) { close(syncChanH2) }).
		Once()

	ctx, loopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.SyncLoop(ctx, make(chan<- error))
		t.Log("SyncLoop exited.")
	}()

	// --- Send H+2 Event First ---
	heightInCh <- daHeightEvent{Header: headerH2, Data: dataH2, DaHeight: daHeight}

	// Wait for H+2 to be cached (but not processed since H+1 is missing)
	require.Eventually(func() bool {
		return m.headerCache.GetItem(heightH2) != nil && m.dataCache.GetItem(heightH2) != nil
	}, 1*time.Second, 10*time.Millisecond, "H+2 header and data should be cached")

	assert.Equal(initialHeight, m.GetLastState().LastBlockHeight, "Height should not have advanced yet")
	assert.NotNil(m.headerCache.GetItem(heightH2), "Header H+2 should be in cache")
	assert.NotNil(m.dataCache.GetItem(heightH2), "Data H+2 should be in cache")

	// --- Send H+1 Event Second ---
	heightInCh <- daHeightEvent{Header: headerH1, Data: dataH1, DaHeight: daHeight}

	t.Log("Waiting for Sync H+1 to complete...")

	// --- Wait for Processing (H+1 then H+2) ---
	select {
	case <-syncChanH1:
		t.Log("Sync H+1 completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for sync H+1 to complete")
	}

	t.Log("Waiting for SyncLoop H+2 to complete...")

	select {
	case <-syncChanH2:
		t.Log("Sync H+2 completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for sync H+2 to complete")
	}

	wg.Wait()

	mockStore.AssertExpectations(t)
	mockExec.AssertExpectations(t)

	finalState := m.GetLastState()
	assert.Equal(expectedStateH2.LastBlockHeight, finalState.LastBlockHeight)
	assert.Equal(expectedStateH2.AppHash, finalState.AppHash)
	assert.Equal(expectedStateH2.LastBlockTime, finalState.LastBlockTime)
	assert.Equal(expectedStateH2.DAHeight, finalState.DAHeight)

	assert.Nil(m.headerCache.GetItem(heightH1), "Header cache should be cleared for H+1")
	assert.Nil(m.dataCache.GetItem(heightH1), "Data cache should be cleared for H+1")
	assert.Nil(m.headerCache.GetItem(heightH2), "Header cache should be cleared for H+2")
	assert.Nil(m.dataCache.GetItem(heightH2), "Data cache should be cleared for H+2")
}

// TestSyncLoop_IgnoreDuplicateEvents verifies that the SyncLoop correctly processes
// a block once even if the header and data events are received multiple times.
func TestSyncLoop_IgnoreDuplicateEvents(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	initialHeight := uint64(40)
	initialState := types.State{
		LastBlockHeight: initialHeight,
		AppHash:         []byte("initial_app_hash_dup"),
		ChainID:         "syncLoopTest",
		DAHeight:        35,
	}
	heightH1 := initialHeight + 1
	daHeight := initialState.DAHeight

	m, mockStore, mockExec, ctx, cancel, heightInCh, _ := setupManagerForSyncLoopTest(t, initialState)
	defer cancel()

	// --- Block H+1 Data ---
	headerH1, dataH1, privKeyH1 := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: heightH1, NTxs: 1}, initialState.ChainID, initialState.AppHash)
	require.NotNil(headerH1)
	require.NotNil(dataH1)
	require.NotNil(privKeyH1)

	appHashH1 := []byte("app_hash_h1_dup")
	expectedStateH1, err := initialState.NextState(headerH1.Header, appHashH1)
	require.NoError(err)

	var txsH1 [][]byte
	for _, tx := range dataH1.Txs {
		txsH1 = append(txsH1, tx)
	}

	syncChanH1 := make(chan struct{})

	// --- Mock Expectations (Expect processing exactly ONCE) ---
	mockExec.On("ExecuteTxs", mock.Anything, txsH1, heightH1, headerH1.Time(), initialState.AppHash).
		Return(appHashH1, uint64(1), nil).Once()
	mockStore.On("SaveBlockData", mock.Anything, headerH1, dataH1, &headerH1.Signature).Return(nil).Once()
	mockStore.On("SetHeight", mock.Anything, heightH1).Return(nil).Once()
	mockStore.On("UpdateState", mock.Anything, expectedStateH1).Return(nil).
		Run(func(args mock.Arguments) { close(syncChanH1) }).
		Once()

	ctx, loopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.SyncLoop(ctx, make(chan<- error))
		t.Log("SyncLoop exited.")
	}()

	// --- Send First Event ---
	heightInCh <- daHeightEvent{Header: headerH1, Data: dataH1, DaHeight: daHeight}

	t.Log("Waiting for first sync to complete...")
	select {
	case <-syncChanH1:
		t.Log("First sync completed.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for first sync to complete")
	}

	// --- Send Duplicate Event ---
	heightInCh <- daHeightEvent{Header: headerH1, Data: dataH1, DaHeight: daHeight}

	// Give the sync loop a chance to process duplicates (if it would)
	// Since we expect no processing, we just wait for the context timeout
	// The mock expectations will fail if duplicates are processed

	wg.Wait()

	// Assertions
	mockStore.AssertExpectations(t) // Crucial: verifies calls happened exactly once
	mockExec.AssertExpectations(t)  // Crucial: verifies calls happened exactly once

	finalState := m.GetLastState()
	assert.Equal(expectedStateH1.LastBlockHeight, finalState.LastBlockHeight)
	assert.Equal(expectedStateH1.AppHash, finalState.AppHash)
	assert.Equal(expectedStateH1.LastBlockTime, finalState.LastBlockTime)
	assert.Equal(expectedStateH1.DAHeight, finalState.DAHeight)

	// Assert caches are cleared
	assert.Nil(m.headerCache.GetItem(heightH1), "Header cache should be cleared for H+1")
	assert.Nil(m.dataCache.GetItem(heightH1), "Data cache should be cleared for H+1")
}

// TestSyncLoop_ErrorOnApplyError verifies that the SyncLoop halts if ApplyBlock fails.
// Halting after sync loop error is handled in full.go and is not tested here.
func TestSyncLoop_ErrorOnApplyError(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	initialHeight := uint64(50)
	initialState := types.State{
		LastBlockHeight: initialHeight,
		AppHash:         []byte("initial_app_hash_panic"),
		ChainID:         "syncLoopTest",
		DAHeight:        45,
	}
	heightH1 := initialHeight + 1
	daHeight := initialState.DAHeight

	m, mockStore, mockExec, ctx, cancel, heightInCh, _ := setupManagerForSyncLoopTest(t, initialState)
	defer cancel() // Ensure context cancellation happens even on panic

	// --- Block H+1 Data ---
	headerH1, dataH1, privKeyH1 := types.GenerateRandomBlockCustomWithAppHash(&types.BlockConfig{Height: heightH1, NTxs: 1}, initialState.ChainID, initialState.AppHash)
	require.NotNil(headerH1)
	require.NotNil(dataH1)
	require.NotNil(privKeyH1)

	var txsH1 [][]byte
	for _, tx := range dataH1.Txs {
		txsH1 = append(txsH1, tx)
	}

	applyErrorSignal := make(chan struct{})
	applyError := errors.New("apply failed")

	// --- Mock Expectations ---
	mockExec.On("ExecuteTxs", mock.Anything, txsH1, heightH1, headerH1.Time(), initialState.AppHash).
		Return(nil, uint64(100), applyError).                       // Return the error that should cause panic
		Run(func(args mock.Arguments) { close(applyErrorSignal) }). // Signal *before* returning error
		Once()
	// NO further calls expected after Apply error

	ctx, loopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer loopCancel()

	var wg sync.WaitGroup
	wg.Add(1)

	errCh := make(chan error, 1)

	go func() {
		defer wg.Done()
		m.SyncLoop(ctx, errCh)
	}()

	// --- Send Event ---
	t.Logf("Sending height event for height %d", heightH1)
	heightInCh <- daHeightEvent{Header: headerH1, Data: dataH1, DaHeight: daHeight}

	t.Log("Waiting for ApplyBlock error...")
	select {
	case <-applyErrorSignal:
		t.Log("ApplyBlock error occurred.")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for ApplyBlock error signal")
	}

	require.Error(<-errCh, "SyncLoop should return an error when ApplyBlock errors")

	wg.Wait()

	mockStore.AssertExpectations(t)
	mockExec.AssertExpectations(t)

	finalState := m.GetLastState()
	assert.Equal(initialState.LastBlockHeight, finalState.LastBlockHeight, "State height should not change after panic")
	assert.Equal(initialState.AppHash, finalState.AppHash, "State AppHash should not change after panic")

	assert.NotNil(m.headerCache.GetItem(heightH1), "Header cache should still contain item for H+1")
	assert.NotNil(m.dataCache.GetItem(heightH1), "Data cache should still contain item for H+1")
}
