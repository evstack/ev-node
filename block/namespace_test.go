package block

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
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
		store:                       mockStore,
		config:                      config.Config{DA: daConfig},
		genesis:                     genesis.Genesis{ProposerAddress: addr},
		daHeight:                    &atomic.Uint64{},
		heightInCh:                  make(chan NewHeightEvent, eventInChLength),
		headerStore:                 headerStore,
		dataStore:                   dataStore,
		headerCache:                 cache.NewCache[types.SignedHeader](),
		dataCache:                   cache.NewCache[types.Data](),
		headerStoreCh:               make(chan struct{}, 1),
		dataStoreCh:                 make(chan struct{}, 1),
		retrieveCh:                  make(chan struct{}, 1),
		daIncluderCh:                make(chan struct{}, 1),
		logger:                      mockLogger,
		lastStateMtx:                &sync.RWMutex{},
		da:                          mockDAClient,
		signer:                      noopSigner,
		metrics:                     NopMetrics(),
		namespaceMigrationCompleted: &atomic.Bool{},
	}

	manager.daHeight.Store(100)
	manager.daIncludedHeight.Store(0)

	t.Cleanup(cancel)

	return manager, mockDAClient, mockStore, cancel
}

// TestProcessNextDAHeaderAndData_MixedResults tests scenarios where header retrieval succeeds but data fails, and vice versa
func TestProcessNextDAHeaderAndData_MixedResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		headerError   bool
		headerMessage string
		dataError     bool
		dataMessage   string
		expectError   bool
		errorContains string
	}{
		{
			name:          "header succeeds, data fails",
			headerError:   false,
			headerMessage: "",
			dataError:     true,
			dataMessage:   "data retrieval failed",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "header fails, data succeeds",
			headerError:   true,
			headerMessage: "header retrieval failed",
			dataError:     false,
			dataMessage:   "",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "header from future, data succeeds",
			headerError:   true,
			headerMessage: "height from future",
			dataError:     false,
			dataMessage:   "",
			expectError:   true,
			errorContains: "height from future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			daConfig := config.DAConfig{
				HeaderNamespace: "test-headers",
				DataNamespace:   "test-data",
			}
			manager, mockDA, _, cancel := setupManagerForNamespaceTest(t, daConfig)
			daManager := newDARetriever(manager)
			defer cancel()

			// Mark migration as completed to skip legacy namespace check
			manager.namespaceMigrationCompleted.Store(true)

			// Set up DA mock expectations
			if tt.headerError {
				// Header namespace fails
				if strings.Contains(tt.headerMessage, "height from future") {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(nil,
						fmt.Errorf("wrapped: %w", coreda.ErrHeightFromFuture)).Once()
				} else {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(nil,
						fmt.Errorf("%s", tt.headerMessage)).Once()
				}
			} else {
				// Header namespace succeeds but returns no data (simulating success but not a valid blob)
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-headers")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			}

			if tt.dataError {
				// Data namespace fails
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(nil,
					fmt.Errorf("%s", tt.dataMessage)).Once()
			} else {
				// Data namespace succeeds but returns no data
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			}

			ctx := context.Background()
			err := daManager.processNextDAHeaderAndData(ctx)

			if tt.expectError {
				require.Error(t, err, "Expected error but got none")
				assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
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
		legacyHasData           bool
		newNamespaceHasData     bool
		expectMigrationComplete bool
		expectLegacyCall        bool
	}{
		{
			name:                    "migration not started, legacy has data",
			initialMigrationState:   false,
			legacyHasData:           true,
			newNamespaceHasData:     false,
			expectMigrationComplete: false,
			expectLegacyCall:        true,
		},
		{
			name:                    "migration not started, no legacy data, new namespace has data",
			initialMigrationState:   false,
			legacyHasData:           false,
			newNamespaceHasData:     true,
			expectMigrationComplete: true,
			expectLegacyCall:        true,
		},
		{
			name:                    "migration not started, no data anywhere",
			initialMigrationState:   false,
			legacyHasData:           false,
			newNamespaceHasData:     false,
			expectMigrationComplete: true,
			expectLegacyCall:        true,
		},
		{
			name:                    "migration already completed",
			initialMigrationState:   true,
			legacyHasData:           true, // shouldn't matter
			newNamespaceHasData:     true,
			expectMigrationComplete: true,
			expectLegacyCall:        false, // should skip legacy check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			daConfig := config.DAConfig{
				Namespace:       "legacy-namespace",
				HeaderNamespace: "test-headers",
				DataNamespace:   "test-data",
			}
			manager, mockDA, _, cancel := setupManagerForNamespaceTest(t, daConfig)
			daManager := newDARetriever(manager)
			defer cancel()

			// Set initial migration state
			manager.namespaceMigrationCompleted.Store(tt.initialMigrationState)

			if tt.expectLegacyCall {
				// Mock legacy namespace call
				if tt.legacyHasData {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("legacy-namespace")).Return(&coreda.GetIDsResult{
						IDs:       []coreda.ID{[]byte("legacy-id")},
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
						IDs:       []coreda.ID{[]byte("header-id")},
						Timestamp: time.Now(),
					}, nil).Once()
					mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("header-id")}, []byte("test-headers")).Return(
						[][]byte{[]byte("header-data")}, nil,
					).Once()

					// Data namespace
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
						IDs:       []coreda.ID{[]byte("data-id")},
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
					IDs:       []coreda.ID{[]byte("header-id")},
					Timestamp: time.Now(),
				}, nil).Once()
				mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("header-id")}, []byte("test-headers")).Return(
					[][]byte{[]byte("header-data")}, nil,
				).Once()

				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte("test-data")).Return(&coreda.GetIDsResult{
					IDs:       []coreda.ID{[]byte("data-id")},
					Timestamp: time.Now(),
				}, nil).Once()
				mockDA.On("Get", mock.Anything, []coreda.ID{[]byte("data-id")}, []byte("test-data")).Return(
					[][]byte{[]byte("data-blob")}, nil,
				).Once()
			}

			// If migration should complete, expect persistence call

			ctx := context.Background()
			err := daManager.processNextDAHeaderAndData(ctx)

			require.NoError(t, err, "processNextDAHeaderAndData should not return error")

			// Verify migration state
			assert.Equal(t, tt.expectMigrationComplete, manager.namespaceMigrationCompleted.Load(),
				"Migration completion state should match expected")

			// Verify migration state based on expected behavior
			// Note: we can't easily verify specific data retrieval without making the test overly complex
			// The main goal is to test that the migration completion logic works

			mockDA.AssertExpectations(t)
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
		store:                       mockStore,
		config:                      config.Config{DA: daConfig},
		genesis:                     genesis.Genesis{ProposerAddress: addr},
		daHeight:                    &atomic.Uint64{},
		headerStore:                 headerStore,
		dataStore:                   dataStore,
		headerCache:                 cache.NewCache[types.SignedHeader](),
		dataCache:                   cache.NewCache[types.Data](),
		logger:                      zerolog.Nop(),
		signer:                      noopSigner,
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
		name                 string
		legacyNamespace      string
		headerNamespace      string
		dataNamespace        string
		expectLegacyFallback bool
		description          string
	}{
		{
			name:                 "only legacy namespace configured",
			legacyNamespace:      "old-namespace",
			headerNamespace:      "",
			dataNamespace:        "",
			expectLegacyFallback: true,
			description:          "When only legacy namespace is set, it acts as both header and data namespace",
		},
		{
			name:                 "all namespaces configured",
			legacyNamespace:      "old-namespace",
			headerNamespace:      "new-headers",
			dataNamespace:        "new-data",
			expectLegacyFallback: true,
			description:          "Should check legacy first, then try new namespaces",
		},
		{
			name:                 "no namespaces configured",
			legacyNamespace:      "",
			headerNamespace:      "",
			dataNamespace:        "",
			expectLegacyFallback: false,
			description:          "Should use default namespaces only",
		},
	}

	for _, tt := range tests {
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
			daManager := newDARetriever(manager)
			defer cancel()

			// Start with migration not completed
			manager.namespaceMigrationCompleted.Store(false)

			// Check if we should expect a legacy namespace check
			// Legacy check happens when migration is not completed and legacy namespace is configured
			if tt.legacyNamespace != "" {
				// When legacy namespace is the same as header/data namespaces,
				// the legacy check still happens first (it's a separate call)
				if tt.legacyNamespace == headerNS && headerNS == dataNS {
					// All three namespaces are the same, so we'll get 3 calls total:
					// 1 for legacy check + 1 for header + 1 for data
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(tt.legacyNamespace)).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Times(3)
				} else if tt.legacyNamespace == headerNS || tt.legacyNamespace == dataNS {
					// Legacy matches one of the new namespaces
					// We'll get the legacy call plus one or two more depending on whether header == data
					totalCalls := 1 // legacy call
					if headerNS == dataNS {
						totalCalls += 2 // header and data are same
					} else {
						totalCalls += 1 // the one that matches legacy
					}
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(tt.legacyNamespace)).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Times(totalCalls)

					// Mock the non-matching namespace if header != data
					if headerNS != dataNS {
						nonMatchingNS := headerNS
						if tt.legacyNamespace == headerNS {
							nonMatchingNS = dataNS
						}
						mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(nonMatchingNS)).Return(&coreda.GetIDsResult{
							IDs: []coreda.ID{},
						}, coreda.ErrBlobNotFound).Once()
					}
				} else {
					// Legacy is different from both header and data
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(tt.legacyNamespace)).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Once()

					// Mock header and data calls
					if headerNS == dataNS {
						mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
							IDs: []coreda.ID{},
						}, coreda.ErrBlobNotFound).Twice()
					} else {
						mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
							IDs: []coreda.ID{},
						}, coreda.ErrBlobNotFound).Once()
						mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(dataNS)).Return(&coreda.GetIDsResult{
							IDs: []coreda.ID{},
						}, coreda.ErrBlobNotFound).Once()
					}
				}
			} else {
				// No legacy namespace configured, just mock the new namespace calls
				if headerNS == dataNS {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Twice()
				} else {
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Once()
					mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(dataNS)).Return(&coreda.GetIDsResult{
						IDs: []coreda.ID{},
					}, coreda.ErrBlobNotFound).Once()
				}
			}

			ctx := context.Background()
			err := daManager.processNextDAHeaderAndData(ctx)

			// Should succeed with no data found (returns nil on StatusNotFound)
			require.NoError(t, err)

			mockDA.AssertExpectations(t)
		})
	}
}
