package syncing

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
)

// DARetriever defines the interface for retrieving events from the DA layer
type DARetriever interface {
	// RetrieveFromDA retrieves blocks from the specified DA height and returns height events
	RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error)
	// ProcessBlobs parses raw blob bytes at a given DA height into height events.
	// Used by the DAFollower to process subscription blobs inline without re-fetching.
	ProcessBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent
}

// daRetriever handles DA retrieval operations for syncing
type daRetriever struct {
	client  da.Client
	cache   cache.CacheManager
	genesis genesis.Genesis
	logger  zerolog.Logger
}

// NewDARetriever creates a new DA retriever
func NewDARetriever(
	client da.Client,
	cache cache.CacheManager,
	genesis genesis.Genesis,
	logger zerolog.Logger,
) *daRetriever {
	return &daRetriever{
		client:  client,
		cache:   cache,
		genesis: genesis,
		logger:  logger.With().Str("component", "da_retriever").Logger(),
	}
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
	return r.processBlobs(blobsResp.Data, daHeight), nil
}

// fetchBlobs retrieves blobs from the DA layer at the specified height
func (r *daRetriever) fetchBlobs(ctx context.Context, daHeight uint64) (datypes.ResultRetrieve, error) {
	res := r.client.RetrieveBlobs(ctx, daHeight, r.client.GetHeaderNamespace())

	switch res.Code {
	case datypes.StatusError:
		return res, fmt.Errorf("DA retrieval failed: %s", res.Message)
	case datypes.StatusHeightFromFuture:
		return res, fmt.Errorf("%w: height from future", datypes.ErrHeightFromFuture)
	case datypes.StatusNotFound:
		return res, fmt.Errorf("%w: blob not found", datypes.ErrBlobNotFound)
	case datypes.StatusSuccess:
		r.logger.Debug().Uint64("da_height", daHeight).Msg("successfully retrieved from DA")
		return res, nil
	default:
		return res, nil
	}
}

// ProcessBlobs processes raw blob bytes to extract headers and data and returns height events.
// This is the public interface used by the DAFollower for inline subscription processing.
func (r *daRetriever) ProcessBlobs(_ context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	return r.processBlobs(blobs, daHeight)
}

// processBlobs processes retrieved blobs to extract headers and data and returns height events
func (r *daRetriever) processBlobs(blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	var events []common.DAHeightEvent

	for _, bz := range blobs {
		if len(bz) == 0 {
			continue
		}

		header, data, envelopeSig, err := common.UnmarshalBlockBlob(bz)
		if err != nil {
			r.logger.Debug().Err(err).Msg("failed to decode combined block blob, skipping")
			continue
		}

		if err := header.Header.ValidateBasic(); err != nil {
			r.logger.Debug().Err(err).Msg("invalid header structure")
			continue
		}

		if err := r.assertExpectedProposer(header.ProposerAddress); err != nil {
			r.logger.Debug().Err(err).Msg("unexpected proposer")
			continue
		}

		if len(envelopeSig) > 0 {
			if header.Signer.PubKey == nil {
				r.logger.Debug().Msg("header signer has no pubkey, cannot verify envelope")
				continue
			}
			payload, err := header.MarshalBinary()
			if err != nil {
				r.logger.Debug().Err(err).Msg("failed to marshal header for verification")
				continue
			}
			if valid, err := header.Signer.PubKey.Verify(payload, envelopeSig); err != nil || !valid {
				r.logger.Info().Err(err).Msg("DA envelope signature verification failed")
				continue
			}
			r.logger.Debug().Uint64("height", header.Height()).Msg("DA envelope signature verified")
		}

		// Update cache with DA inclusion information
		headerHash := header.MemoizeHash().String()
		r.cache.SetHeaderDAIncluded(headerHash, daHeight, header.Height())

		if len(data.Txs) > 0 {
			dataHash := data.DACommitment().String()
			r.cache.SetDataDAIncluded(dataHash, daHeight, data.Height())
		}

		r.logger.Debug().
			Str("header_hash", headerHash).
			Uint64("da_height", daHeight).
			Uint64("height", header.Height()).
			Msg("decoded combined block blob")

		events = append(events, common.DAHeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: daHeight,
			Source:   common.SourceDA,
		})
	}

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

// assertExpectedProposer validates the proposer address
func (r *daRetriever) assertExpectedProposer(proposerAddr []byte) error {
	return assertExpectedProposer(r.genesis, proposerAddr)
}
