package single

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

func TestLoad_MigratesLegacyKeys(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	// Use an in-memory datastore
	memdb := ds.NewMapDatastore()
	prefix := "batching" // Prefix used by the queue

	// 1. Setup DB with mixed data: valid keys and legacy keys
	// Valid key (seq 0x8000...00)
	validSeq := initialSeqNum
	validKey := seqToKey(validSeq) // 16-char hex
	validBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("valid-tx")}}

	// Persist valid batch manually
	pbValid := &pb.Batch{Txs: validBatch.Transactions}
	validBytes, _ := proto.Marshal(pbValid)
	// Note: NewBatchQueue uses "batching" prefix, so keys are /batching/<key>
	err := memdb.Put(ctx, ds.NewKey("/"+prefix+"/"+validKey), validBytes)
	require.NoError(err)

	// Legacy key (just a random hash string, not 16 hex chars)
	// Let's use 32 bytes hex encoded -> 64 chars
	legacyHash := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	legacyBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("legacy-tx")}}

	pbLegacy := &pb.Batch{Txs: legacyBatch.Transactions}
	legacyBytes, _ := proto.Marshal(pbLegacy)
	err = memdb.Put(ctx, ds.NewKey("/"+prefix+"/"+legacyHash), legacyBytes)
	require.NoError(err)

	// Another legacy key (to test multiple)
	legacyHash2 := "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00"
	legacyBatch2 := coresequencer.Batch{Transactions: [][]byte{[]byte("legacy-tx-2")}}
	pbLegacy2 := &pb.Batch{Txs: legacyBatch2.Transactions}
	legacyBytes2, _ := proto.Marshal(pbLegacy2)
	err = memdb.Put(ctx, ds.NewKey("/"+prefix+"/"+legacyHash2), legacyBytes2)
	require.NoError(err)

	// 2. Create Queue and call Load
	bq := NewBatchQueue(memdb, prefix, 0)
	err = bq.Load(ctx)
	require.NoError(err)

	// 3. Verify Queue State
	// 1 valid + 2 legacy = 3 items total
	require.Equal(3, bq.Size())

	// Check Order:
	// Appended items should be LAST.
	// Valid strings are < Appended strings (because we use nextAddSeq which is > validSeq)
	// So Next() should return Valid item first.
	// Between legacy items, the order depends on how they were iterated from DB and appended.
	// Iteration order: L2 (11...), L1 (aa...)
	// Process L2: append -> [Valid, L2]
	// Process L1: append -> [Valid, L2, L1]

	// So expected retrieval order: Valid, L2, L1.

	// 1st Item (Valid)
	item1, err := bq.Next(ctx)
	require.NoError(err)
	require.Equal(validBatch.Transactions, item1.Transactions)

	// 2nd Item (L2)
	item2, err := bq.Next(ctx)
	require.NoError(err)
	require.Equal(legacyBatch2.Transactions, item2.Transactions)

	// 3rd Item (L1)
	item3, err := bq.Next(ctx)
	require.NoError(err)
	require.Equal(legacyBatch.Transactions, item3.Transactions)

	// Queue empty
	require.Equal(0, bq.Size())

	// 4. Verify DB State
	// Old legacy keys should be gone from the DB (under the prefix).
	// New seq keys should exist.

	q := query.Query{Prefix: "/" + prefix}
	results, err := memdb.Query(ctx, q)
	require.NoError(err)
	defer results.Close()

	keys := make([]string, 0)
	for range results.Next() {
		keys = append(keys, "found")
	}

	// We expect 0 keys in DB because we called Next() 3 times which deletes them from DB as well!
	require.Empty(keys, "Expected DB to be empty after processing all items")
}

func TestLoad_Migration_DBCheck(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	memdb := ds.NewMapDatastore()
	prefix := "batching"

	// Setup valid existing item as well to verify it stays before legacy
	validSeq := initialSeqNum
	validKey := seqToKey(validSeq)
	validBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("valid-data")}}
	pbValid := &pb.Batch{Txs: validBatch.Transactions}
	validBytes, _ := proto.Marshal(pbValid)
	memdb.Put(ctx, ds.NewKey("/"+prefix+"/"+validKey), validBytes)

	// Setup legacy key
	legacyHash := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	legacyBatch := coresequencer.Batch{Transactions: [][]byte{[]byte("legacy-data")}}
	pbLegacy := &pb.Batch{Txs: legacyBatch.Transactions}
	legacyBytes, _ := proto.Marshal(pbLegacy)
	memdb.Put(ctx, ds.NewKey("/"+prefix+"/"+legacyHash), legacyBytes)

	// Load
	bq := NewBatchQueue(memdb, prefix, 0)
	require.NoError(bq.Load(ctx))

	// Verify DB keys
	q := query.Query{Prefix: "/" + prefix}
	results, err := memdb.Query(ctx, q)
	require.NoError(err)
	defer results.Close()

	foundLegacy := false
	foundValid := false
	for res := range results.Next() {
		k := ds.NewKey(res.Key).Name()
		if k == legacyHash {
			t.Errorf("Legacy key %s still exists in DB", legacyHash)
		}
		// Check if it's the expected migrated key (initialSeqNum + 1)
		// validSeq = initialSeqNum. So legacy should be at initialSeqNum + 1.
		expectedKey := seqToKey(initialSeqNum + 1)
		if k == expectedKey {
			foundLegacy = true
			var pbCheck pb.Batch
			proto.Unmarshal(res.Value, &pbCheck)
			require.Equal(legacyBatch.Transactions, pbCheck.Txs)
		}
		if k == validKey {
			foundValid = true
		}
	}
	require.True(foundValid, "Valid key should persist")
	require.True(foundLegacy, "Migrated key not found in DB")
}
