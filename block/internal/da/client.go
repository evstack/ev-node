package da

import (
	"context"

	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Config contains configuration for the unified DA client.
type Config struct {
	// Client is the underlying DA client implementation (node or app)
	Client                   datypes.BlobClient
	Logger                   zerolog.Logger
	Namespace                string
	DataNamespace            string
	ForcedInclusionNamespace string
}

// client wraps any datypes.BlobClient with namespace handling.
// It is unexported; callers should use the exported Client interface.
type client struct {
	daClient           datypes.BlobClient
	logger             zerolog.Logger
	namespaceBz        []byte
	dataNamespaceBz    []byte
	forcedNamespaceBz  []byte
	hasForcedNamespace bool
}

// Ensure client implements the FullClient interface.
var _ FullClient = (*client)(nil)

// NewClient creates a new unified DA client wrapper.
func NewClient(cfg Config) FullClient {
	if cfg.Client == nil {
		return nil
	}

	hasForcedNamespace := cfg.ForcedInclusionNamespace != ""
	var forcedNamespaceBz []byte
	if hasForcedNamespace {
		forcedNamespaceBz = datypes.NamespaceFromString(cfg.ForcedInclusionNamespace).Bytes()
	}

	return &client{
		daClient:           cfg.Client,
		logger:             cfg.Logger.With().Str("component", "da_client").Logger(),
		namespaceBz:        datypes.NamespaceFromString(cfg.Namespace).Bytes(),
		dataNamespaceBz:    datypes.NamespaceFromString(cfg.DataNamespace).Bytes(),
		forcedNamespaceBz:  forcedNamespaceBz,
		hasForcedNamespace: hasForcedNamespace,
	}
}

// Submit submits blobs to the DA layer.
func (c *client) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	return c.daClient.Submit(ctx, data, gasPrice, namespace, options)
}

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
func (c *client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	return c.daClient.Retrieve(ctx, height, namespace)
}

// RetrieveForcedInclusion retrieves blobs from the forced inclusion namespace at the specified height.
func (c *client) RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve {
	if !c.hasForcedNamespace {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "forced inclusion namespace not configured",
				Height:  height,
			},
		}
	}
	return c.Retrieve(ctx, height, c.forcedNamespaceBz)
}

// Get retrieves blobs by their IDs.
func (c *client) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	return c.daClient.Get(ctx, ids, namespace)
}

// GetLatestDAHeight returns the latest height available on the DA layer.
func (c *client) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	return c.daClient.GetLatestDAHeight(ctx)
}

// GetHeaderNamespace returns the header namespace bytes.
func (c *client) GetHeaderNamespace() []byte {
	return c.namespaceBz
}

// GetDataNamespace returns the data namespace bytes.
func (c *client) GetDataNamespace() []byte {
	return c.dataNamespaceBz
}

// GetForcedInclusionNamespace returns the forced inclusion namespace bytes.
func (c *client) GetForcedInclusionNamespace() []byte {
	return c.forcedNamespaceBz
}

// HasForcedInclusionNamespace reports whether forced inclusion namespace is configured.
func (c *client) HasForcedInclusionNamespace() bool {
	return c.hasForcedNamespace
}

// GetProofs returns inclusion proofs for the provided IDs.
func (c *client) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	return c.daClient.GetProofs(ctx, ids, namespace)
}

// Validate validates commitments against the corresponding proofs.
func (c *client) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	return c.daClient.Validate(ctx, ids, proofs, namespace)
}
