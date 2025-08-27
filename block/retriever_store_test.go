package block

import (
	// ... other necessary imports ...
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/signer/noop"
	storepkg "github.com/evstack/ev-node/pkg/store"

	// Use existing store mock if available, or define one
	mocksStore "github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

func setupManagerForStoreRetrieveTest(t *testing.T) (
	m *Manager,
	mockStore *mocksStore.MockStore,
	mockHeaderStore *extmocks.MockStore[*types.SignedHeader],
	mockDataStore *extmocks.MockStore[*types.Data],
	headerStoreCh chan struct{},
	dataStoreCh chan struct{},
	heightInCh chan daHeightEvent,
	ctx context.Context,
	cancel context.CancelFunc,
) {
	t.Helper()

	// Mocks
	mockStore = mocksStore.NewMockStore(t)
	mockHeaderStore = extmocks.NewMockStore[*types.SignedHeader](t)
	mockDataStore = extmocks.NewMockStore[*types.Data](t)

	// Channels (buffered to prevent deadlocks in simple test cases)
	headerStoreCh = make(chan struct{}, 1)
	dataStoreCh = make(chan struct{}, 1)
	heightInCh = make(chan daHeightEvent, 10)

	// Config & Genesis
	nodeConf := config.DefaultConfig
	genDoc, pk, _ := types.GetGenesisWithPrivkey("test") // Use test helper

	logger := zerolog.Nop()
	ctx, cancel = context.WithCancel(context.Background())

	// Mock initial metadata reads during manager creation if necessary
	mockStore.On("GetMetadata", mock.Anything, storepkg.DAIncludedHeightKey).Return(nil, ds.ErrNotFound).Maybe()
	mockStore.On("GetMetadata", mock.Anything, storepkg.LastBatchDataKey).Return(nil, ds.ErrNotFound).Maybe()

	signer, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)
	// Create Manager instance with mocks and necessary fields
	m = &Manager{
		store:         mockStore,
		headerStore:   mockHeaderStore,
		dataStore:     mockDataStore,
		headerStoreCh: headerStoreCh,
		dataStoreCh:   dataStoreCh,
		heightInCh:    heightInCh,
		logger:        logger,
		genesis:       genDoc,
		daHeight:      &atomic.Uint64{},
		lastStateMtx:  new(sync.RWMutex),
		config:        nodeConf,
		signer:        signer,
	}

	// initialize da included height
	if height, err := m.store.GetMetadata(ctx, storepkg.DAIncludedHeightKey); err == nil && len(height) == 8 {
		m.daIncludedHeight.Store(binary.LittleEndian.Uint64(height))
	}

	return m, mockStore, mockHeaderStore, mockDataStore, headerStoreCh, dataStoreCh, heightInCh, ctx, cancel
}

// TestDataStoreRetrieveLoop_RetrievesNewData verifies that the data store retrieve loop retrieves new data correctly.
func TestDataStoreRetrieveLoop_RetrievesNewData(t *testing.T) {
	assert := assert.New(t)
	m, mockStore, mockHeaderStore, mockDataStore, _, dataStoreCh, heightInCh, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	initialHeight := uint64(5)
	mockStore.On("Height", ctx).Return(initialHeight, nil).Maybe()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.DataStoreRetrieveLoop(ctx)
	}()

	// Configure mock
	newHeight := uint64(6)

	// Generate a consistent header and data pair using test utilities
	blockConfig := types.BlockConfig{
		Height:       newHeight,
		NTxs:         1,
		ProposerAddr: m.genesis.ProposerAddress,
	}
	expectedHeader, expectedData, _ := types.GenerateRandomBlockCustom(&blockConfig, m.genesis.ChainID)

	// Set the signer address to match the proposer address
	signerAddr, err := m.signer.GetAddress()
	require.NoError(t, err)
	signerPubKey, err := m.signer.GetPublic()
	require.NoError(t, err)
	expectedHeader.Signer.Address = signerAddr
	expectedHeader.Signer.PubKey = signerPubKey

	// Re-sign the header with our test signer to make it valid
	headerBytes, err := expectedHeader.Header.MarshalBinary()
	require.NoError(t, err)
	sig, err := m.signer.Sign(headerBytes)
	require.NoError(t, err)
	expectedHeader.Signature = sig

	mockDataStore.On("Height").Return(newHeight).Once() // Height check after trigger
	mockDataStore.On("GetByHeight", ctx, newHeight).Return(expectedData, nil).Once()
	mockHeaderStore.On("GetByHeight", ctx, newHeight).Return(expectedHeader, nil).Once()

	// Trigger the loop
	dataStoreCh <- struct{}{}

	// Verify data received
	select {
	case receivedEvent := <-heightInCh:
		assert.Equal(expectedData, receivedEvent.Data)
		assert.Equal(expectedHeader, receivedEvent.Header)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for height event on heightInCh")
	}

	// Cancel context and wait for loop to finish
	cancel()
	wg.Wait()

	// Assert mock expectations
	mockDataStore.AssertExpectations(t)
	mockHeaderStore.AssertExpectations(t)
}

// TestDataStoreRetrieveLoop_RetrievesMultipleData verifies that the data store retrieve loop retrieves multiple new data entries.
func TestDataStoreRetrieveLoop_RetrievesMultipleData(t *testing.T) {
	assert := assert.New(t)
	m, mockStore, mockHeaderStore, mockDataStore, _, dataStoreCh, heightInCh, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	initialHeight := uint64(5)
	mockStore.On("Height", ctx).Return(initialHeight, nil).Maybe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.DataStoreRetrieveLoop(ctx)
	}()

	// Configure mock
	finalHeight := uint64(8) // Retrieve heights 6, 7, 8
	expectedData := make(map[uint64]*types.Data)
	expectedHeaders := make(map[uint64]*types.SignedHeader)

	// Get signer info for creating valid headers
	signerAddr, err := m.signer.GetAddress()
	require.NoError(t, err)
	signerPubKey, err := m.signer.GetPublic()
	require.NoError(t, err)

	for h := initialHeight + 1; h <= finalHeight; h++ {
		// Generate consistent header and data pair
		blockConfig := types.BlockConfig{
			Height:       h,
			NTxs:         1,
			ProposerAddr: m.genesis.ProposerAddress,
		}
		header, data, _ := types.GenerateRandomBlockCustom(&blockConfig, m.genesis.ChainID)

		// Set proper signer info and re-sign
		header.Signer.Address = signerAddr
		header.Signer.PubKey = signerPubKey
		headerBytes, err := header.Header.MarshalBinary()
		require.NoError(t, err)
		sig, err := m.signer.Sign(headerBytes)
		require.NoError(t, err)
		header.Signature = sig

		expectedData[h] = data
		expectedHeaders[h] = header
	}

	mockDataStore.On("Height").Return(finalHeight).Once()
	for h := initialHeight + 1; h <= finalHeight; h++ {
		mockDataStore.On("GetByHeight", mock.Anything, h).Return(expectedData[h], nil).Once()
		mockHeaderStore.On("GetByHeight", mock.Anything, h).Return(expectedHeaders[h], nil).Once()
	}

	// Trigger the loop
	dataStoreCh <- struct{}{}

	// Verify data received
	receivedCount := 0
	expectedCount := len(expectedData)
	timeout := time.After(2 * time.Second)
	for receivedCount < expectedCount {
		select {
		case receivedEvent := <-heightInCh:
			receivedCount++
			h := receivedEvent.Data.Height()
			assert.Contains(expectedData, h)
			assert.Equal(expectedData[h], receivedEvent.Data)
			assert.Equal(expectedHeaders[h], receivedEvent.Header)
			expectedItem, ok := expectedData[h]
			assert.True(ok, "Received unexpected height: %d", h)
			if ok {
				assert.Equal(expectedItem, receivedEvent.Data)
				delete(expectedData, h)
			}
		case <-timeout:
			t.Fatalf("timed out waiting for all height events on heightInCh, received %d out of %d", receivedCount, int(finalHeight-initialHeight))
		}
	}
	assert.Empty(expectedData, "Not all expected data items were received")

	// Cancel context and wait for loop to finish
	cancel()
	wg.Wait()

	// Assert mock expectations
	mockDataStore.AssertExpectations(t)
}

// TestDataStoreRetrieveLoop_NoNewData verifies that the data store retrieve loop handles the case where there is no new data.
func TestDataStoreRetrieveLoop_NoNewData(t *testing.T) {
	m, mockStore, _, mockDataStore, _, dataStoreCh, _, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	currentHeight := uint64(5)
	mockStore.On("Height", ctx).Return(currentHeight, nil).Once()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.DataStoreRetrieveLoop(ctx)
	}()

	mockDataStore.On("Height").Return(currentHeight).Once()

	dataStoreCh <- struct{}{}

	select {
	case receivedEvent := <-m.heightInCh:
		t.Fatalf("received unexpected height event on heightInCh: %+v", receivedEvent)
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	wg.Wait()

	mockDataStore.AssertExpectations(t)
}

// TestDataStoreRetrieveLoop_HandlesFetchError verifies that the data store retrieve loop handles fetch errors gracefully.
func TestDataStoreRetrieveLoop_HandlesFetchError(t *testing.T) {
	m, mockStore, _, mockDataStore, _, dataStoreCh, _, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	currentHeight := uint64(5)
	mockStore.On("Height", ctx).Return(currentHeight, nil).Once()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.DataStoreRetrieveLoop(ctx)
	}()

	newHeight := uint64(6)
	fetchError := errors.New("failed to fetch data")

	mockDataStore.On("Height").Return(newHeight).Once()
	mockDataStore.On("GetByHeight", mock.Anything, newHeight).Return(nil, fetchError).Once()

	dataStoreCh <- struct{}{}

	// Verify no events received
	select {
	case receivedEvent := <-m.heightInCh:
		t.Fatalf("received unexpected height event on heightInCh: %+v", receivedEvent)
	case <-time.After(100 * time.Millisecond):
		// Expected behavior: no events since heights are the same
	}

	cancel()
	wg.Wait()

	mockDataStore.AssertExpectations(t)
}

// TestHeaderStoreRetrieveLoop_RetrievesNewHeader verifies that the header store retrieve loop retrieves new headers correctly.
func TestHeaderStoreRetrieveLoop_RetrievesNewHeader(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	m, mockStore, mockHeaderStore, mockDataStore, headerStoreCh, _, heightInCh, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	initialHeight := uint64(0)
	newHeight := uint64(1)

	mockStore.On("Height", ctx).Return(initialHeight, nil).Maybe()

	validHeader, err := types.GetFirstSignedHeader(m.signer, m.genesis.ChainID)
	require.NoError(err)
	require.Equal(m.genesis.ProposerAddress, validHeader.ProposerAddress)

	validData := &types.Data{Metadata: &types.Metadata{Height: newHeight}}

	mockHeaderStore.On("Height").Return(newHeight).Once() // Height check after trigger
	mockHeaderStore.On("GetByHeight", mock.Anything, newHeight).Return(validHeader, nil).Once()
	mockDataStore.On("GetByHeight", ctx, newHeight).Return(validData, nil).Once()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.HeaderStoreRetrieveLoop(ctx)
	}()

	headerStoreCh <- struct{}{}

	select {
	case receivedEvent := <-heightInCh:
		assert.Equal(validHeader, receivedEvent.Header)
		assert.Equal(validData, receivedEvent.Data)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for height event on heightInCh")
	}

	cancel()
	wg.Wait()

	mockHeaderStore.AssertExpectations(t)
}

// TestHeaderStoreRetrieveLoop_RetrievesMultipleHeaders verifies that the header store retrieve loop retrieves multiple new headers.
func TestHeaderStoreRetrieveLoop_RetrievesMultipleHeaders(t *testing.T) {
	// Test enabled - fixed to work with new architecture
	assert := assert.New(t)
	require := require.New(t)

	m, mockStore, mockHeaderStore, mockDataStore, headerStoreCh, _, heightInCh, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	initialHeight := uint64(5)
	finalHeight := uint64(8)
	numHeaders := finalHeight - initialHeight

	headers := make([]*types.SignedHeader, numHeaders)
	expectedData := make(map[uint64]*types.Data)

	// Get signer info for creating valid headers
	signerAddr, err := m.signer.GetAddress()
	require.NoError(err)
	signerPubKey, err := m.signer.GetPublic()
	require.NoError(err)

	for i := uint64(0); i < numHeaders; i++ {
		currentHeight := initialHeight + 1 + i

		// Generate consistent header and data pair
		blockConfig := types.BlockConfig{
			Height:       currentHeight,
			NTxs:         1,
			ProposerAddr: m.genesis.ProposerAddress,
		}
		h, data, _ := types.GenerateRandomBlockCustom(&blockConfig, m.genesis.ChainID)

		// Set proper signer info and re-sign
		h.Signer.Address = signerAddr
		h.Signer.PubKey = signerPubKey
		headerBytes, err := h.Header.MarshalBinary()
		require.NoError(err)
		sig, err := m.signer.Sign(headerBytes)
		require.NoError(err)
		h.Signature = sig

		headers[i] = h
		expectedData[currentHeight] = data

		mockHeaderStore.On("GetByHeight", ctx, currentHeight).Return(h, nil).Once()
		mockDataStore.On("GetByHeight", ctx, currentHeight).Return(expectedData[currentHeight], nil).Once()
	}

	mockHeaderStore.On("Height").Return(finalHeight).Once()

	mockStore.On("Height", ctx).Return(initialHeight, nil).Maybe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.HeaderStoreRetrieveLoop(ctx)
	}()

	headerStoreCh <- struct{}{}

	receivedCount := 0
	timeout := time.After(3 * time.Second)
	expectedHeaders := make(map[uint64]*types.SignedHeader)
	for _, h := range headers {
		expectedHeaders[h.Height()] = h
	}

	for receivedCount < int(numHeaders) {
		select {
		case receivedEvent := <-heightInCh:
			receivedCount++
			h := receivedEvent.Header
			expected, found := expectedHeaders[h.Height()]
			assert.True(found, "Received unexpected header height: %d", h.Height())
			if found {
				assert.Equal(expected, h)
				assert.Equal(expectedData[h.Height()], receivedEvent.Data)
				delete(expectedHeaders, h.Height()) // Remove found header
			}
		case <-timeout:
			t.Fatalf("timed out waiting for all height events on heightInCh, received %d out of %d", receivedCount, numHeaders)
		}
	}

	assert.Empty(expectedHeaders, "Not all expected headers were received")

	// Cancel context and wait for loop to finish
	cancel()
	wg.Wait()

	// Verify mock expectations
	mockHeaderStore.AssertExpectations(t)
}

// TestHeaderStoreRetrieveLoop_NoNewHeaders verifies that the header store retrieve loop handles the case where there are no new headers.
func TestHeaderStoreRetrieveLoop_NoNewHeaders(t *testing.T) {
	m, mockStore, mockHeaderStore, _, headerStoreCh, _, _, ctx, cancel := setupManagerForStoreRetrieveTest(t)
	defer cancel()

	currentHeight := uint64(5)

	mockStore.On("Height", ctx).Return(currentHeight, nil).Once()
	mockHeaderStore.On("Height").Return(currentHeight).Once()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.HeaderStoreRetrieveLoop(ctx)
	}()

	// Trigger the loop
	headerStoreCh <- struct{}{}

	// Wait briefly and assert nothing is received
	select {
	case receivedEvent := <-m.heightInCh:
		t.Fatalf("received unexpected height event on heightInCh: %+v", receivedEvent)
	case <-time.After(100 * time.Millisecond):
		// Expected timeout, nothing received
	}

	// Cancel context and wait for loop to finish
	cancel()
	wg.Wait()

	// Verify mock expectations
	mockHeaderStore.AssertExpectations(t)
}
