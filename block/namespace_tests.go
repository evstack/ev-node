package block

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	goheaderstore "github.com/celestiaorg/go-header/store"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/cache"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/signer/noop"
	storepkg "github.com/evstack/ev-node/pkg/store"
	rollmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// setupManagerForNamespaceTest creates a Manager with mocked DA and store for testing namespace functionality
func setupManagerForNamespaceTest(t *testing.T, daConfig config.DAConfig) (*Manager, *rollmocks.MockDA, *rollmocks.MockStore, context.CancelFunc) {
	t.Helper()
	mockDAClient := rollmocks.NewMockDA(t)
	mockStore := rollmocks.NewMockStore(t)
	mockLogger := zerolog.Nop()

	headerStore, _ := goheaderstore.NewStore[*types.SignedHeader](ds.NewMapDatastore())
	dataStore, _ := goheaderstore.NewStore[*types.Data](ds.NewMapDatastore())

	// Set up basic mocks
	mockStore.On("GetState", mock.Anything).Return(types.State{DAHeight: 100}, nil).Maybe()
	mockStore.On("SetHeight", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockStore.On("SetMetadata", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockStore.On("GetMetadata", mock.Anything, storepkg.DAIncludedHeightKey).Return([]byte{}, ds.ErrNotFound).Maybe()

	// Create temp directory for persistence testing
	tempDir, err := os.MkdirTemp("", "namespace-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Mock the persistence file operations
	mockStore.On("GetMetadata", mock.Anything, namespaceMigrationKey).Return([]byte{}, ds.ErrNotFound).Maybe()
	mockStore.On("SetMetadata", mock.Anything, namespaceMigrationKey, mock.Anything).Return(nil).Maybe()

	_, cancel := context.WithCancel(context.Background())

	// Create a mock signer
	src := rand.Reader
	pk, _, err := crypto.GenerateEd25519Key(src)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)

	addr, err := noopSigner.GetAddress()
	require.NoError(t, err)

	manager := &Manager{
		store:         mockStore,
		config:        config.Config{DA: daConfig},
		genesis:       genesis.Genesis{ProposerAddress: addr},
		daHeight:      &atomic.Uint64{},
		headerInCh:    make(chan NewHeaderEvent, eventInChLength),
		headerStore:   headerStore,
		dataInCh:      make(chan NewDataEvent, eventInChLength),
		dataStore:     dataStore,
		headerCache:   cache.NewCache[types.SignedHeader](),
		dataCache:     cache.NewCache[types.Data](),
		headerStoreCh: make(chan struct{}, 1),
		dataStoreCh:   make(chan struct{}, 1),
		retrieveCh:    make(chan struct{}, 1),
		daIncluderCh:  make(chan struct{}, 1),
		logger:        mockLogger,
		da:            mockDAClient,
		signer:        noopSigner,
		metrics:       NopMetrics(),
		namespaceMigrationCompleted: &atomic.Bool{},
	}

	manager.daHeight.Store(100)
	t.Cleanup(cancel)

	return manager, mockDAClient, mockStore, cancel
}

// TestFetchBlobs_MixedResults tests scenarios where header retrieval succeeds but data fails, and vice versa
func TestFetchBlobs_MixedResults(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		headerStatus   coreda.StatusCode
		headerMessage  string
		dataStatus     coreda.StatusCode  
		dataMessage    string
		expectedStatus coreda.StatusCode
		expectError    bool
		errorContains  string
	}{
		{
			name:           "header succeeds, data fails",
			headerStatus:   coreda.StatusSuccess,
			headerMessage:  "",
			dataStatus:     coreda.StatusError,
			dataMessage:    "data retrieval failed",
			expectedStatus: coreda.StatusError,
			expectError:    true,
			errorContains:  "data retrieval failed",
		},
		{
			name:           "header fails, data succeeds", 
			headerStatus:   coreda.StatusError,
			headerMessage:  "header retrieval failed",
			dataStatus:     coreda.StatusSuccess,
			dataMessage:    "",
			expectedStatus: coreda.StatusError,
			expectError:    true,
			errorContains:  "header retrieval failed",
		},
		{
			name:           "both succeed with some data",
			headerStatus:   coreda.StatusSuccess,
			headerMessage:  "",
			dataStatus:     coreda.StatusSuccess,
			dataMessage:    "",
			expectedStatus: coreda.StatusSuccess,
			expectError:    false,
		},
		{
			name:           "header from future, data succeeds",
			headerStatus:   coreda.StatusHeightFromFuture,
			headerMessage:  "height from future",
			dataStatus:     coreda.StatusSuccess,
			dataMessage:    "",
			expectedStatus: coreda.StatusHeightFromFuture,
			expectError:    true,
			errorContains:  "height from future",
		},
		{
			name:           "header succeeds, data from future",
			headerStatus:   coreda.StatusSuccess,
			headerMessage:  "",
			dataStatus:     coreda.StatusHeightFromFuture,
			dataMessage:    "height from future",
			expectedStatus: coreda.StatusHeightFromFuture,
			expectError:    true,
			errorContains:  "height from future",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			daConfig := config.DAConfig{
				HeaderNamespace: "test-headers",
				DataNamespace:   "test-data",
			}
			manager, mockDA, _, cancel := setupManagerForNamespaceTest(t, daConfig)
			defer cancel()

			// Mark migration as completed to skip legacy namespace check
			manager.namespaceMigrationCompleted.Store(true)

			// Mock header namespace retrieval
			headerResult := &coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:    tt.headerStatus,
					Message: tt.headerMessage,
				},
			}
			if tt.headerStatus == coreda.StatusSuccess {
				headerResult.Data = [][]byte{[]byte("header-data")}
				headerResult.IDs = []coreda.ID{[]byte("header-id")}
			}

			// Mock data namespace retrieval
			dataResult := &coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:    tt.dataStatus,
					Message: tt.dataMessage,
				},
			}
			if tt.dataStatus == coreda.StatusSuccess {
				dataResult.Data = [][]byte{[]byte("data-blob")}
				dataResult.IDs = []coreda.ID{[]byte("data-id")}
			}

			// Set up DA mock expectations for GetIDs and Get calls
			if tt.headerStatus == coreda.StatusSuccess {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{[]byte("header-id")},
					Timestamp: time.Now(),
				}, nil).Once()
				mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("header-id")}, []byte("test-headers")).Return(
					[][]byte{[]byte("header-data")}, nil,
				).Once()
			} else if tt.headerStatus == coreda.StatusError {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(nil, 
					fmt.Errorf(tt.headerMessage)).Once()
			} else if tt.headerStatus == coreda.StatusHeightFromFuture {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(nil,
					fmt.Errorf("wrapped: %w", coreda.ErrHeightFromFuture)).Once()
			} else {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			}

			if tt.dataStatus == coreda.StatusSuccess {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{[]byte("data-id")},
					Timestamp: time.Now(),
				}, nil).Once()
				mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("data-id")}, []byte("test-data")).Return(
					[][]byte{[]byte("data-blob")}, nil,
				).Once()
			} else if tt.dataStatus == coreda.StatusError {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(nil,
					fmt.Errorf(tt.dataMessage)).Once()
			} else if tt.dataStatus == coreda.StatusHeightFromFuture {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(nil,
					fmt.Errorf("wrapped: %w", coreda.ErrHeightFromFuture)).Once()
			} else {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			}

			ctx := context.Background()
			result, err := manager.fetchBlobs(ctx, 100)

			assert.Equal(t, tt.expectedStatus, result.Code, "Status code should match expected")

			if tt.expectError {
				require.Error(t, err, "Expected error but got none")
				assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
				if tt.headerStatus == coreda.StatusSuccess && tt.dataStatus == coreda.StatusSuccess {
					assert.Len(t, result.Data, 2, "Should have data from both namespaces")
					assert.Contains(t, result.Data, []byte("header-data"))
					assert.Contains(t, result.Data, []byte("data-blob"))
				}
			}

			mockDA.AssertExpectations(t)
		})
	}
}

// TestNamespaceMigration_Completion tests the migration completion logic and persistence
func TestNamespaceMigration_Completion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		initialMigrationState   bool
		legacyHasData          bool
		newNamespaceHasData    bool
		expectMigrationComplete bool
		expectLegacyCall       bool
	}{
		{
			name:                    "migration not started, legacy has data",
			initialMigrationState:   false,
			legacyHasData:          true,
			newNamespaceHasData:    false,
			expectMigrationComplete: false,
			expectLegacyCall:       true,
		},
		{
			name:                    "migration not started, no legacy data, new namespace has data",
			initialMigrationState:   false,
			legacyHasData:          false,
			newNamespaceHasData:    true,
			expectMigrationComplete: true,
			expectLegacyCall:       true,
		},
		{
			name:                    "migration not started, no data anywhere",
			initialMigrationState:   false,
			legacyHasData:          false,
			newNamespaceHasData:    false,
			expectMigrationComplete: true,
			expectLegacyCall:       true,
		},
		{
			name:                    "migration already completed",
			initialMigrationState:   true,
			legacyHasData:          true, // shouldn't matter
			newNamespaceHasData:    true,
			expectMigrationComplete: true,
			expectLegacyCall:       false, // should skip legacy check
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			daConfig := config.DAConfig{
				Namespace:       "legacy-namespace",
				HeaderNamespace: "test-headers",
				DataNamespace:   "test-data",
			}
			manager, mockDA, mockStore, cancel := setupManagerForNamespaceTest(t, daConfig)
			defer cancel()

			// Set initial migration state
			manager.namespaceMigrationCompleted.Store(tt.initialMigrationState)

			if tt.expectLegacyCall {
				// Mock legacy namespace call
				if tt.legacyHasData {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("legacy-namespace")).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{[]byte("legacy-id")},
						Timestamp: time.Now(),
					}, nil).Once()
					mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("legacy-id")}, []byte("legacy-namespace")).Return(
						[][]byte{[]byte("legacy-data")}, nil,
					).Once()
				} else {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("legacy-namespace")).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Once()
				}
			}

			if !tt.legacyHasData && tt.expectLegacyCall {
				// Mock new namespace calls
				if tt.newNamespaceHasData {
					// Header namespace
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{[]byte("header-id")},
						Timestamp: time.Now(),
					}, nil).Once()
					mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("header-id")}, []byte("test-headers")).Return(
						[][]byte{[]byte("header-data")}, nil,
					).Once()

					// Data namespace
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{[]byte("data-id")},
						Timestamp: time.Now(),
					}, nil).Once()
					mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("data-id")}, []byte("test-data")).Return(
						[][]byte{[]byte("data-blob")}, nil,
					).Once()
				} else {
					// Both namespaces return not found
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Once()
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Once()
				}
			} else if !tt.expectLegacyCall {
				// Migration already completed, should only call new namespaces
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{[]byte("header-id")},
					Timestamp: time.Now(),
				}, nil).Once()
				mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("header-id")}, []byte("test-headers")).Return(
					[][]byte{[]byte("header-data")}, nil,
				).Once()

				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{[]byte("data-id")},
					Timestamp: time.Now(),
				}, nil).Once()
				mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("data-id")}, []byte("test-data")).Return(
					[][]byte{[]byte("data-blob")}, nil,
				).Once()
			}

			// If migration should complete, expect persistence call
			if tt.expectMigrationComplete && !tt.initialMigrationState {
				mockStore.On("SetMetadata", mock.Anything, namespaceMigrationKey, []byte{1}).Return(nil).Once()
			}

			ctx := context.Background()
			result, err := manager.fetchBlobs(ctx, 100)

			require.NoError(t, err, "fetchBlobs should not return error")

			// Verify migration state
			assert.Equal(t, tt.expectMigrationComplete, manager.namespaceMigrationCompleted.Load(),
				"Migration completion state should match expected")

			if tt.legacyHasData && tt.expectLegacyCall {
				// Should have used legacy data
				assert.Equal(t, coreda.StatusSuccess, result.Code)
				assert.Contains(t, result.Data, []byte("legacy-data"))
			} else if tt.newNamespaceHasData {
				// Should have used new namespace data
				assert.Equal(t, coreda.StatusSuccess, result.Code)
				assert.Contains(t, result.Data, []byte("header-data"))
				assert.Contains(t, result.Data, []byte("data-blob"))
			} else if !tt.expectLegacyCall {
				// Migration completed, using new namespaces
				assert.Equal(t, coreda.StatusSuccess, result.Code)
			} else {
				// No data found anywhere
				assert.Equal(t, coreda.StatusNotFound, result.Code)
			}

			mockDA.AssertExpectations(t)
			mockStore.AssertExpectations(t)
		})
	}
}

// TestNamespaceMigration_PersistenceReload tests that migration state survives restart
func TestNamespaceMigration_PersistenceReload(t *testing.T) {
	t.Parallel()

	daConfig := config.DAConfig{
		Namespace:       "legacy-namespace",
		HeaderNamespace: "test-headers", 
		DataNamespace:   "test-data",
	}

	// Simulate completed migration persisted to disk
	mockStore := rollmocks.NewMockStore(t)
	mockStore.On("GetState", mock.Anything).Return(types.State{DAHeight: 100}, nil).Maybe()
	mockStore.On("GetMetadata", mock.Anything, namespaceMigrationKey).Return([]byte{1}, nil).Once() // Migration completed

	headerStore, _ := goheaderstore.NewStore[*types.SignedHeader](ds.NewMapDatastore())
	dataStore, _ := goheaderstore.NewStore[*types.Data](ds.NewMapDatastore())

	src := rand.Reader
	pk, _, err := crypto.GenerateEd25519Key(src)
	require.NoError(t, err)
	noopSigner, err := noop.NewNoopSigner(pk)
	require.NoError(t, err)

	addr, err := noopSigner.GetAddress()
	require.NoError(t, err)

	manager := &Manager{
		store:         mockStore,
		config:        config.Config{DA: daConfig},
		genesis:       genesis.Genesis{ProposerAddress: addr},
		daHeight:      &atomic.Uint64{},
		headerStore:   headerStore,
		dataStore:     dataStore,
		headerCache:   cache.NewCache[types.SignedHeader](),
		dataCache:     cache.NewCache[types.Data](),
		logger:        zerolog.Nop(),
		signer:        noopSigner,
		namespaceMigrationCompleted: &atomic.Bool{},
	}

	// Initialize migration state from persistence (simulates restart)
	ctx := context.Background()
	migrationCompleted, err := manager.loadNamespaceMigrationState(ctx)
	require.NoError(t, err)
	assert.True(t, migrationCompleted, "Migration should be loaded as completed from persistence")

	manager.namespaceMigrationCompleted.Store(migrationCompleted)
	assert.True(t, manager.namespaceMigrationCompleted.Load(), "Manager should reflect completed migration state")

	mockStore.AssertExpectations(t)
}

// TestLegacyNamespaceDetection tests the legacy namespace fallback behavior
func TestLegacyNamespaceDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		legacyNamespace    string
		headerNamespace    string
		dataNamespace      string
		expectLegacyFallback bool
		description        string
	}{
		{
			name:               "only legacy namespace configured",
			legacyNamespace:    "old-namespace",
			headerNamespace:    "",
			dataNamespace:      "",
			expectLegacyFallback: false, // Should use defaults
			description:        "When only legacy namespace is set, should use default new namespaces",
		},
		{
			name:               "all namespaces configured",
			legacyNamespace:    "old-namespace",
			headerNamespace:    "new-headers",
			dataNamespace:      "new-data",
			expectLegacyFallback: true,
			description:        "Should check legacy first, then try new namespaces",
		},
		{
			name:               "no namespaces configured",
			legacyNamespace:    "",
			headerNamespace:    "",
			dataNamespace:      "",
			expectLegacyFallback: false,
			description:        "Should use default namespaces only",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			daConfig := config.DAConfig{
				Namespace:       tt.legacyNamespace,
				HeaderNamespace: tt.headerNamespace,
				DataNamespace:   tt.dataNamespace,
			}

			// Test the GetHeaderNamespace and GetDataNamespace methods
			headerNS := daConfig.GetHeaderNamespace()
			dataNS := daConfig.GetDataNamespace()

			if tt.headerNamespace != "" {
				assert.Equal(t, tt.headerNamespace, headerNS)
			} else if tt.legacyNamespace != "" {
				assert.Equal(t, tt.legacyNamespace, headerNS)
			} else {
				assert.Equal(t, "rollkit-headers", headerNS) // Default
			}

			if tt.dataNamespace != "" {
				assert.Equal(t, tt.dataNamespace, dataNS)
			} else if tt.legacyNamespace != "" {
				assert.Equal(t, tt.legacyNamespace, dataNS)
			} else {
				assert.Equal(t, "rollkit-data", dataNS) // Default
			}

			// Test actual behavior in fetchBlobs
			manager, mockDA, _, cancel := setupManagerForNamespaceTest(t, daConfig)
			defer cancel()

			// Start with migration not completed
			manager.namespaceMigrationCompleted.Store(false)

			if tt.expectLegacyFallback && tt.legacyNamespace != "" {
				// Should try legacy namespace first
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(tt.legacyNamespace)).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			}

			// Then should try new namespaces
			mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
				IDs: []coreda.ID{},
			}, coreda.ErrBlobNotFound).Once()

			mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(dataNS)).Return(&coreda.GetIDsResult{
				IDs: []coreda.ID{},
			}, coreda.ErrBlobNotFound).Once()

			ctx := context.Background()
			result, err := manager.fetchBlobs(ctx, 100)

			require.NoError(t, err)
			assert.Equal(t, coreda.StatusNotFound, result.Code)

			mockDA.AssertExpectations(t)
		})
	}
}