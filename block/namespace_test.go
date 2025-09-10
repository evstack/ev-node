package block

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

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
		store:         mockStore,
		config:        config.Config{DA: daConfig},
		genesis:       genesis.Genesis{ProposerAddress: addr},
		daHeight:      &atomic.Uint64{},
		heightInCh:    make(chan daHeightEvent, eventInChLength),
		headerStore:   headerStore,
		dataStore:     dataStore,
		headerCache:   cache.NewCache[types.SignedHeader](),
		dataCache:     cache.NewCache[types.Data](),
		headerStoreCh: make(chan struct{}, 1),
		dataStoreCh:   make(chan struct{}, 1),
		retrieveCh:    make(chan struct{}, 1),
		daIncluderCh:  make(chan struct{}, 1),
		logger:        mockLogger,
		lastStateMtx:  &sync.RWMutex{},
		da:            mockDAClient,
		signer:        noopSigner,
		metrics:       NopMetrics(),
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
				Namespace:     "test-headers",
				DataNamespace: "test-data",
			}
			manager, mockDA, _, cancel := setupManagerForNamespaceTest(t, daConfig)
			daManager := newDARetriever(manager)
			defer cancel()

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

// TestLegacyNamespaceDetection tests the legacy namespace fallback behavior
func TestLegacyNamespaceDetection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		namespace     string
		dataNamespace string
	}{
		{
			// When only legacy namespace is set, it acts as both header and data namespace
			name:          "only legacy namespace configured",
			namespace:     "namespace",
			dataNamespace: "",
		},
		{
			// Should check both namespaces. It will behave as legacy when data namespace is empty
			name:          "all namespaces configured",
			namespace:     "old-namespace",
			dataNamespace: "new-data",
		},
		{
			// Should use default namespaces only
			name:          "no namespaces configured",
			namespace:     "",
			dataNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			daConfig := config.DAConfig{
				Namespace:     tt.namespace,
				DataNamespace: tt.dataNamespace,
			}

			// Test the GetNamespace and GetDataNamespace methods
			headerNS := daConfig.GetNamespace()
			dataNS := daConfig.GetDataNamespace()

			if tt.namespace != "" {
				assert.Equal(t, tt.namespace, headerNS)
			} else {
				assert.Equal(t, "", headerNS) // Default
			}

			if tt.dataNamespace != "" {
				assert.Equal(t, tt.dataNamespace, dataNS)
			} else if tt.namespace != "" {
				assert.Equal(t, tt.namespace, dataNS)
			} else {
				assert.Equal(t, "", dataNS)
			}

			// Test actual behavior in fetchBlobs
			manager, mockDA, _, cancel := setupManagerForNamespaceTest(t, daConfig)
			daManager := newDARetriever(manager)
			defer cancel()

			// When  namespace is the same as data namespaces,
			// only one call is expected
			if headerNS == dataNS {
				// All 2 namespaces are the same, so we'll get 1 call.
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			} else {
				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(headerNS)).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()

				mockDA.On("GetIDs", mock.Anything, uint64(100), []byte(dataNS)).Return(&coreda.GetIDsResult{
					IDs: []coreda.ID{},
				}, coreda.ErrBlobNotFound).Once()
			}

			err := daManager.processNextDAHeaderAndData(context.Background())
			// Should succeed with no data found (returns nil on StatusNotFound)
			require.NoError(t, err)

			mockDA.AssertExpectations(t)
		})
	}
}
