package syncing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// DARetriever defines the interface for retrieving events from the DA layer
type DARetriever interface {
	// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
	RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error)
	// ProcessBlobs parses raw blob bytes at a given DA height into height events.
	// Used by the DAFollower to process subscription blobs inline without re-fetching.
	ProcessBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent
}

type pendingDataCleaner interface {
	removePendingData(height uint64)
}

type expectedProposerProvider func(height uint64) ([]byte, bool)

// syncedHeaderProvider returns the stored header for a height the node has
// already synced. synced is false for heights above the node's height. The
// header is nil when the height is synced but no longer stored.
type syncedHeaderProvider func(height uint64) (header *types.SignedHeader, synced bool, err error)

type pendingHeader struct {
	header   *types.SignedHeader
	daHeight uint64
	// envelope is set when the DA envelope signature verified against the header's signer key.
	envelope bool
}

type pendingData struct {
	data     *types.Data
	hash     types.Hash
	signer   []byte
	daHeight uint64
}

// daRetriever handles DA retrieval operations for syncing
type daRetriever struct {
	client  da.Client
	cache   cache.CacheManager
	genesis genesis.Genesis
	logger  zerolog.Logger

	expectedProposer  expectedProposerProvider
	syncedHeader      syncedHeaderProvider
	signatureProvider types.SyncNodeSignatureBytesProvider

	mu sync.Mutex
	// Anyone can post to the DA namespaces, so every decoded blob is only a
	// candidate: it is not marked DA included and does not displace another
	// blob until it is authenticated. Candidates for future heights are kept
	// without a cap; growing them costs one paid blob per entry, and a cap
	// would let an outsider crowd out the header of a newly rotated proposer.
	// Transient: rebuilt from DA on restart.
	pendingHeaders map[uint64][]pendingHeader
	pendingData    map[uint64][]pendingData

	// strictMode indicates if the node has seen an authenticated DAHeaderEnvelope
	// and should now reject all legacy/unsigned headers.
	strictMode bool
}

// NewDARetriever creates a new DA retriever
func NewDARetriever(
	client da.Client,
	cache cache.CacheManager,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *daRetriever {
	return &daRetriever{
		client:         client,
		cache:          cache,
		genesis:        genesis,
		logger:         logger.With().Str("component", "da_retriever").Logger(),
		pendingHeaders: make(map[uint64][]pendingHeader),
		pendingData:    make(map[uint64][]pendingData),
		strictMode:     false,
	}
}

func (r *daRetriever) setExpectedProposerProvider(provider expectedProposerProvider) {
	r.expectedProposer = provider
}

func (r *daRetriever) setSyncedHeaderProvider(provider syncedHeaderProvider) {
	r.syncedHeader = provider
}

func (r *daRetriever) setSignatureProvider(provider types.SyncNodeSignatureBytesProvider) {
	r.signatureProvider = provider
}

// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
func (r *daRetriever) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
	r.logger.Debug().Uint64("da_height", daHeight).Msg("retrieving from DA")
	blobsResp, err := r.fetchBlobs(ctx, daHeight)
	if err != nil {
		return nil, err
	}

	// Check for context cancellation upfront
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.logger.Debug().Int("blobs", len(blobsResp.Data)).Uint64("da_height", daHeight).Msg("retrieved blob data")
	return r.processBlobs(ctx, blobsResp.Data, daHeight), nil
}

// fetchBlobs retrieves blobs from both header and data namespaces
func (r *daRetriever) fetchBlobs(ctx context.Context, daHeight uint64) (datypes.ResultRetrieve, error) {
	// Retrieve from both namespaces using the DA client
	headerRes := r.client.RetrieveBlobs(ctx, daHeight, r.client.GetHeaderNamespace())

	// If namespaces are the same, return header result
	if bytes.Equal(r.client.GetHeaderNamespace(), r.client.GetDataNamespace()) {
		return headerRes, r.validateBlobResponse(headerRes, daHeight)
	}

	dataRes := r.client.RetrieveBlobs(ctx, daHeight, r.client.GetDataNamespace())

	// Validate responses
	headerErr := r.validateBlobResponse(headerRes, daHeight)
	// ignoring error not found, as data can have data
	if headerErr != nil && !errors.Is(headerErr, datypes.ErrBlobNotFound) {
		return headerRes, headerErr
	}

	dataErr := r.validateBlobResponse(dataRes, daHeight)
	// ignoring error not found, as header can have data
	if dataErr != nil && !errors.Is(dataErr, datypes.ErrBlobNotFound) {
		return dataRes, dataErr
	}

	// Combine successful results
	combinedResult := datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:   datypes.StatusSuccess,
			Height: daHeight,
		},
		Data: make([][]byte, 0),
	}

	if headerRes.Code == datypes.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, headerRes.Data...)
		combinedResult.IDs = append(combinedResult.IDs, headerRes.IDs...)
	}

	if dataRes.Code == datypes.StatusSuccess {
		combinedResult.Data = append(combinedResult.Data, dataRes.Data...)
		combinedResult.IDs = append(combinedResult.IDs, dataRes.IDs...)
	}

	// Re-throw error not found if both were not found.
	if len(combinedResult.Data) == 0 && len(combinedResult.IDs) == 0 {
		r.logger.Debug().Uint64("da_height", daHeight).Msg("no blob data found")
		combinedResult.Code = datypes.StatusNotFound
		combinedResult.Message = datypes.ErrBlobNotFound.Error()
		return combinedResult, datypes.ErrBlobNotFound
	}

	return combinedResult, nil
}

// validateBlobResponse validates a blob response from DA layer
// those are the only error code returned by da.RetrieveWithHelpers
func (r *daRetriever) validateBlobResponse(res datypes.ResultRetrieve, daHeight uint64) error {
	switch res.Code {
	case datypes.StatusError:
		return fmt.Errorf("DA retrieval failed: %s", res.Message)
	case datypes.StatusHeightFromFuture:
		return fmt.Errorf("%w: height from future", datypes.ErrHeightFromFuture)
	case datypes.StatusNotFound:
		return fmt.Errorf("%w: blob not found", datypes.ErrBlobNotFound)
	case datypes.StatusSuccess:
		r.logger.Debug().Uint64("da_height", daHeight).Msg("successfully retrieved from DA")
		return nil
	default:
		return nil
	}
}

// ProcessBlobs processes raw blob bytes to extract headers and data and returns height events.
// This is the public interface used by the DAFollower for inline subscription processing.
func (r *daRetriever) ProcessBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	return r.processBlobs(ctx, blobs, daHeight)
}

// processBlobs processes retrieved blobs to extract headers and data and returns height events
func (r *daRetriever) processBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, bz := range blobs {
		if len(bz) == 0 {
			continue
		}

		if header, envelope := r.tryDecodeHeader(bz); header != nil {
			r.addPendingHeader(pendingHeader{header: header, daHeight: daHeight, envelope: envelope})
			continue
		}

		if data, signer := r.tryDecodeData(bz); data != nil {
			r.addPendingData(pendingData{data: data, hash: data.Hash(), signer: signer, daHeight: daHeight})
		}
	}

	events := r.settleSyncedHeights(ctx, daHeight)
	events = append(events, r.matchPending(ctx, daHeight)...)

	if len(events) > 0 {
		startHeight := events[0].Header.Height()
		endHeight := events[0].Header.Height()
		for _, event := range events {
			h := event.Header.Height()
			if h < startHeight {
				startHeight = h
			}
			if h > endHeight {
				endHeight = h
			}
		}
		r.logger.Info().Uint64("da_height", daHeight).Uint64("start_height", startHeight).Uint64("end_height", endHeight).Msg("processed blocks from DA")
	}

	return events
}

func (r *daRetriever) addPendingHeader(candidate pendingHeader) {
	height := candidate.header.Height()
	for _, existing := range r.pendingHeaders[height] {
		if existing.envelope == candidate.envelope && sameSignedHeader(existing.header, candidate.header) {
			return
		}
	}
	r.pendingHeaders[height] = append(r.pendingHeaders[height], candidate)
}

func (r *daRetriever) addPendingData(candidate pendingData) {
	height := candidate.data.Height()
	for _, existing := range r.pendingData[height] {
		if bytes.Equal(existing.hash, candidate.hash) && bytes.Equal(existing.signer, candidate.signer) {
			return
		}
	}
	r.pendingData[height] = append(r.pendingData[height], candidate)
}

// settleSyncedHeights resolves candidates for heights the node already holds.
// The stored block is the reference: only the blobs that carry it are marked
// DA included, everything else at that height is dropped. A complete block is
// still returned as an event, which the syncer skips as already processed.
func (r *daRetriever) settleSyncedHeights(ctx context.Context, daHeight uint64) []common.DAHeightEvent {
	if r.syncedHeader == nil {
		return nil
	}

	var events []common.DAHeightEvent
	settle := func(height uint64) {
		stored, synced, err := r.syncedHeader(height)
		if err != nil {
			// keep the candidates and retry on the next pass
			r.logger.Warn().Err(err).Uint64("height", height).Msg("failed to load synced header")
			return
		}
		if !synced {
			return
		}
		if stored != nil {
			for _, candidate := range r.pendingData[height] {
				if types.Validate(stored, candidate.data) == nil {
					r.markDataDAIncluded(candidate)
				}
			}
			for _, candidate := range r.pendingHeaders[height] {
				if !sameSignedHeader(candidate.header, stored) {
					continue
				}
				r.markHeaderDAIncluded(candidate)
				if data, ok := r.dataFor(ctx, candidate.header); ok {
					events = append(events, common.DAHeightEvent{
						Header:   candidate.header,
						Data:     data.data,
						DaHeight: daHeight,
						Source:   common.SourceDA,
					})
				}
				break
			}
		}
		delete(r.pendingHeaders, height)
		delete(r.pendingData, height)
	}

	for height := range r.pendingHeaders {
		settle(height)
	}
	for height := range r.pendingData {
		settle(height)
	}
	return events
}

// matchPending emits an event for every future height that has a header from
// the expected proposer together with the data that header commits to.
func (r *daRetriever) matchPending(ctx context.Context, daHeight uint64) []common.DAHeightEvent {
	events := make([]common.DAHeightEvent, 0, len(r.pendingHeaders))

	for height, candidates := range r.pendingHeaders {
		// Left over from a failed load or synced since settleSyncedHeights ran:
		// it is settled against the stored block on the next pass.
		if r.isSynced(height) {
			continue
		}
		expected := r.expectedProposerFor(height)

		for i, candidate := range candidates {
			header := candidate.header
			if !isFromProposer(header, expected) {
				continue
			}

			data, ok := r.dataFor(ctx, header)
			if !ok {
				// keep header in pending headers until data lands
				r.logger.Debug().Uint64("height", height).Msg("header found but no matching data")
				continue
			}

			// The envelope signature already proves the proposer signed this header.
			if !candidate.envelope && !r.hasValidSignature(header, data.data) {
				r.logger.Debug().Uint64("height", height).Msg("invalid header signature")
				continue
			}

			r.markHeaderDAIncluded(candidate)
			if !isEmptyDataExpected(header) {
				r.markDataDAIncluded(data)
			}

			events = append(events, common.DAHeightEvent{
				Header:   header,
				Data:     data.data,
				DaHeight: daHeight,
				Source:   common.SourceDA,
			})

			if len(candidates) == 1 {
				delete(r.pendingHeaders, height)
			} else {
				r.pendingHeaders[height] = append(candidates[:i:i], candidates[i+1:]...)
			}
			break
		}
	}

	return events
}

// dataFor returns the pending data the header commits to. The commitment
// does not cover all metadata, so a copy signed by the header's proposer wins
// over one signed by anyone else.
func (r *daRetriever) dataFor(ctx context.Context, header *types.SignedHeader) (pendingData, bool) {
	var match *pendingData
	for i, candidate := range r.pendingData[header.Height()] {
		if types.Validate(header, candidate.data) != nil {
			continue
		}
		if bytes.Equal(candidate.signer, header.ProposerAddress) {
			return candidate, true
		}
		if match == nil {
			match = &r.pendingData[header.Height()][i]
		}
	}
	if match != nil {
		return *match, true
	}
	if isEmptyDataExpected(header) {
		return pendingData{data: createEmptyDataForHeader(ctx, header)}, true
	}
	return pendingData{}, false
}

func (r *daRetriever) isSynced(height uint64) bool {
	if r.syncedHeader == nil {
		return false
	}
	_, synced, err := r.syncedHeader(height)
	return synced || err != nil
}

// expectedProposerFor returns the proposer expected at a future height, or nil when unknown.
func (r *daRetriever) expectedProposerFor(height uint64) []byte {
	if r.expectedProposer != nil {
		if expected, ok := r.expectedProposer(height); ok && len(expected) > 0 {
			return expected
		}
	}
	return r.genesis.ProposerAddress
}

// sameSignedHeader compares the signature and signer as well as the hash:
// the hash covers neither, so anyone can repost a real header under their own key.
func sameSignedHeader(a, b *types.SignedHeader) bool {
	return bytes.Equal(a.Hash(), b.Hash()) &&
		bytes.Equal(a.Signature, b.Signature) &&
		bytes.Equal(signerAddress(a), signerAddress(b))
}

func signerAddress(header *types.SignedHeader) []byte {
	if header.Signer.PubKey == nil {
		return nil
	}
	return types.KeyAddress(header.Signer.PubKey)
}

// isFromProposer checks the key the header is signed with, not only the
// ProposerAddress field, which anyone can fill in.
func isFromProposer(header *types.SignedHeader, expected []byte) bool {
	if len(expected) == 0 {
		return true
	}
	if header.Signer.PubKey == nil {
		return false
	}
	return bytes.Equal(header.ProposerAddress, expected) && bytes.Equal(signerAddress(header), expected)
}

func (r *daRetriever) hasValidSignature(header *types.SignedHeader, data *types.Data) bool {
	if r.signatureProvider != nil {
		header.SetCustomVerifierForSyncNode(r.signatureProvider)
	}
	return header.ValidateBasicWithData(data) == nil
}

func (r *daRetriever) markHeaderDAIncluded(candidate pendingHeader) {
	header := candidate.header
	headerHash := header.MemoizeHash().String()
	r.cache.SetHeaderDAIncluded(headerHash, candidate.daHeight, header.Height())

	r.logger.Debug().
		Str("header_hash", headerHash).
		Uint64("da_height", candidate.daHeight).
		Uint64("height", header.Height()).
		Msg("header marked as DA included")

	if candidate.envelope && !r.strictMode {
		r.logger.Info().Uint64("height", header.Height()).Msg("valid DA envelope detected, switching to STRICT MODE")
		r.strictMode = true
	}
}

func (r *daRetriever) markDataDAIncluded(candidate pendingData) {
	dataHash := candidate.data.DACommitment().String()
	r.cache.SetDataDAIncluded(dataHash, candidate.daHeight, candidate.data.Height())

	r.logger.Debug().
		Str("data_hash", dataHash).
		Uint64("da_height", candidate.daHeight).
		Uint64("height", candidate.data.Height()).
		Msg("data marked as DA included")
}

func (r *daRetriever) removePendingData(height uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.pendingData, height)
}

// tryDecodeHeader attempts to decode a blob as a header. The second result
// reports whether a DA envelope signature verified against the header's signer key.
func (r *daRetriever) tryDecodeHeader(bz []byte) (*types.SignedHeader, bool) {
	header := new(types.SignedHeader)

	isValidEnvelope := false

	// Attempt to unmarshal as DAHeaderEnvelope and get the envelope signature
	if envelopeSignature, err := header.UnmarshalDAEnvelope(bz); err != nil {
		// If in strict mode, we REQUIRE an envelope.
		if r.strictMode {
			r.logger.Warn().Err(err).Msg("strict mode is enabled, rejecting non-envelope blob")
			return nil, false
		}

		// Fallback for backward compatibility (only if NOT in strict mode)
		r.logger.Debug().Msg("trying legacy decoding")
		var headerPb pb.SignedHeader
		if errLegacy := proto.Unmarshal(bz, &headerPb); errLegacy != nil {
			return nil, false
		}
		if errLegacy := header.FromProto(&headerPb); errLegacy != nil {
			return nil, false
		}
	} else {
		// We have a structurally valid envelope (or at least it parsed)
		if len(envelopeSignature) > 0 {
			if header.Signer.PubKey == nil {
				r.logger.Debug().Msg("header signer has no pubkey, cannot verify envelope")
				return nil, false
			}
			payload, err := header.MarshalBinary()
			if err != nil {
				r.logger.Debug().Err(err).Msg("failed to marshal header for verification")
				return nil, false
			}
			if valid, err := header.Signer.PubKey.Verify(payload, envelopeSignature); err != nil || !valid {
				r.logger.Info().Err(err).Msg("DA envelope signature verification failed")
				return nil, false
			}
			r.logger.Debug().Uint64("height", header.Height()).Msg("DA envelope signature verified")
			isValidEnvelope = true
		}
	}
	if r.strictMode && !isValidEnvelope {
		// no need to print warnings, as tryDecodeHeader could try to decode data first, which will show this warning.
		r.logger.Debug().Msg("strict mode: rejecting block that is not a fully valid envelope")
		return nil, false
	}

	if err := header.Header.ValidateBasic(); err != nil {
		r.logger.Debug().Err(err).Msg("invalid header structure")
		return nil, false
	}

	return header, isValidEnvelope
}

// tryDecodeData attempts to decode a blob as signed data. The second result is
// the address of the key that signed it.
func (r *daRetriever) tryDecodeData(bz []byte) (*types.Data, []byte) {
	var signedData types.SignedData
	if err := signedData.UnmarshalBinary(bz); err != nil {
		return nil, nil
	}

	// Skip completely empty data
	if len(signedData.Txs) == 0 && len(signedData.Signature) == 0 {
		return nil, nil
	}

	if err := assertValidSignedData(&signedData, r.genesis); err != nil {
		r.logger.Debug().Err(err).Msg("invalid signed data")
		return nil, nil
	}

	// Reject data without metadata before using accessors (DACommitment,
	// Height) that assume it is present.
	if signedData.Metadata == nil {
		r.logger.Debug().Msg("invalid signed data: missing metadata")
		return nil, nil
	}

	return &signedData.Data, types.KeyAddress(signedData.Signer.PubKey)
}

// isEmptyDataExpected checks if empty data is expected for a header
func isEmptyDataExpected(header *types.SignedHeader) bool {
	return len(header.DataHash) == 0 || bytes.Equal(header.DataHash, common.DataHashForEmptyTxs)
}

// createEmptyDataForHeader creates empty data for a header
func createEmptyDataForHeader(_ context.Context, header *types.SignedHeader) *types.Data {
	return &types.Data{
		Txs: make(types.Txs, 0),
		Metadata: &types.Metadata{
			ChainID:      header.ChainID(),
			Height:       header.Height(),
			Time:         header.BaseHeader.Time,
			LastDataHash: nil, // LastDataHash must be filled in the syncer, as it is not available here since block n-1 has not been processed yet.
		},
	}
}
