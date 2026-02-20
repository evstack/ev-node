package node

import (
	"context"
	"time"
)

// Header contains the fields from celestia-node's header.ExtendedHeader that we need.
// We only extract the Time field for timestamp determinism.
type Header struct {
	Header    RawHeader `json:"header"`
	Commit    Commit    `json:"commit"`
	DAH       DAHeader  `json:"dah"`
	Height    uint64    `json:"height,string,omitempty"`
	LastHash  []byte    `json:"last_header_hash,omitempty"`
	ChainID   string    `json:"chain_id,omitempty"`
	BlockTime time.Time `json:"time"`
}

// RawHeader contains the raw tendermint header fields.
type RawHeader struct {
	ChainID string    `json:"chain_id"`
	Height  string    `json:"height"`
	Time    time.Time `json:"time"`
}

// Commit contains commit information.
type Commit struct {
	Height string `json:"height"`
}

// DAHeader contains the Data Availability header.
type DAHeader struct {
	RowRoots    [][]byte `json:"row_roots"`
	ColumnRoots [][]byte `json:"column_roots"`
}

// Time returns the block time from the header.
func (h *Header) Time() time.Time {
	// Prefer the nested header time if available
	if !h.Header.Time.IsZero() {
		return h.Header.Time
	}
	return h.BlockTime
}

// HeaderAPI mirrors celestia-node's header module.
// jsonrpc.NewClient wires Internal.* to RPC stubs.
type HeaderAPI struct {
	Internal struct {
		GetByHeight func(
			context.Context,
			uint64,
		) (*Header, error) `perm:"read"`
		LocalHead func(
			context.Context,
		) (*Header, error) `perm:"read"`
		NetworkHead func(
			context.Context,
		) (*Header, error) `perm:"read"`
	}
}

// GetByHeight retrieves a header at the specified height.
func (api *HeaderAPI) GetByHeight(ctx context.Context, height uint64) (*Header, error) {
	return api.Internal.GetByHeight(ctx, height)
}

// LocalHead retrieves the locally synced head header.
func (api *HeaderAPI) LocalHead(ctx context.Context) (*Header, error) {
	return api.Internal.LocalHead(ctx)
}

// NetworkHead retrieves the network head header.
func (api *HeaderAPI) NetworkHead(ctx context.Context) (*Header, error) {
	return api.Internal.NetworkHead(ctx)
}
