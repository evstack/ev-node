package store

import (
	"strconv"

	"github.com/evstack/ev-node/types"
)

const (
	// GenesisDAHeightKey is the key used for persisting the first DA included height in store.
	// It avoids to walk over the HeightToDAHeightKey to find the first DA included height.
	GenesisDAHeightKey = "gdh"

	// HeightToDAHeightKey is the key prefix used for persisting the mapping from a Evolve height
	// to the DA height where the block's header/data was included.
	// Full keys are like: rhb/<evolve_height>/h and rhb/<evolve_height>/d
	HeightToDAHeightKey = "rhb"

	// DAIncludedHeightKey is the key used for persisting the da included height in store.
	DAIncludedHeightKey = "d"

	// LastSubmittedHeaderHeightKey is the key used for persisting the last submitted header height in store.
	LastSubmittedHeaderHeightKey = "last-submitted-header-height"

	// LastPrunedBlockHeightKey is the metadata key used for persisting the last
	// pruned block height in the store.
	LastPrunedBlockHeightKey = "lst-prnd-b"

	// LastPrunedStateHeightKey is the metadata key used for persisting the last
	// pruned state height in the store.
	LastPrunedStateHeightKey = "lst-prnd-s"

	headerPrefix    = "h"
	dataPrefix      = "d"
	signaturePrefix = "c"
	statePrefix     = "s"
	metaPrefix      = "m"
	indexPrefix     = "i"
	heightPrefix    = "t"
)

// GetHeaderKey returns the store key for a block header at the given height.
func GetHeaderKey(height uint64) string {
	return GenerateKey([]string{headerPrefix, strconv.FormatUint(height, 10)})
}

func getHeaderKey(height uint64) string { return GetHeaderKey(height) }

// GetDataKey returns the store key for block data at the given height.
func GetDataKey(height uint64) string {
	return GenerateKey([]string{dataPrefix, strconv.FormatUint(height, 10)})
}

func getDataKey(height uint64) string { return GetDataKey(height) }

// GetSignatureKey returns the store key for a block signature at the given height.
func GetSignatureKey(height uint64) string {
	return GenerateKey([]string{signaturePrefix, strconv.FormatUint(height, 10)})
}

func getSignatureKey(height uint64) string { return GetSignatureKey(height) }

func getStateAtHeightKey(height uint64) string {
	return GenerateKey([]string{statePrefix, strconv.FormatUint(height, 10)})
}

// GetMetaKey returns the store key for a metadata entry.
func GetMetaKey(key string) string {
	return GenerateKey([]string{metaPrefix, key})
}

// GetIndexKey returns the store key for indexing a block by its hash.
func GetIndexKey(hash types.Hash) string {
	return GenerateKey([]string{indexPrefix, hash.String()})
}

func getIndexKey(hash types.Hash) string { return GetIndexKey(hash) }

func getHeightKey() string {
	return GenerateKey([]string{heightPrefix})
}

// GetHeightToDAHeightHeaderKey returns the metadata key for storing the DA height
// where a block's header was included for a given sequencer height.
func GetHeightToDAHeightHeaderKey(height uint64) string {
	return HeightToDAHeightKey + "/" + strconv.FormatUint(height, 10) + "/h"
}

// GetHeightToDAHeightDataKey returns the metadata key for storing the DA height
// where a block's data was included for a given sequencer height.
func GetHeightToDAHeightDataKey(height uint64) string {
	return HeightToDAHeightKey + "/" + strconv.FormatUint(height, 10) + "/d"
}
