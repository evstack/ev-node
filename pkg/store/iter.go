package store

import (
    "context"
    "fmt"
    "sort"
    "strconv"
    "strings"

    dsq "github.com/ipfs/go-datastore/query"

    "github.com/evstack/ev-node/types"
)

// DataAfterHeight returns all Data items with height greater than the provided height,
// sorted by height ascending. It uses the underlying datastore iterator to reduce
// the overhead of N individual Get calls.
func (s *DefaultStore) DataAfterHeight(ctx context.Context, afterHeight uint64) ([]*types.Data, error) {
    // Query all data entries (prefix "/d"). Keys look like "/d/<height>".
    prefix := GenerateKey([]string{dataPrefix}) // "/d"
    results, err := s.db.Query(ctx, dsq.Query{Prefix: prefix})
    if err != nil {
        return nil, fmt.Errorf("datastore query failed: %w", err)
    }
    defer results.Close()

    type pair struct {
        h uint64
        d *types.Data
    }
    items := make([]pair, 0, 1024)

    // Expect keys in the form "/d/<height>"
    keyPrefix := prefix + "/"
    for res := range results.Next() {
        if res.Error != nil {
            return nil, fmt.Errorf("datastore iteration error: %w", res.Error)
        }
        // Fast reject if key doesn't match
        if !strings.HasPrefix(res.Key, keyPrefix) {
            continue
        }
        // Parse height from key
        hs := strings.TrimPrefix(res.Key, keyPrefix)
        height, err := strconv.ParseUint(hs, 10, 64)
        if err != nil {
            // Skip malformed keys rather than failing the entire query
            continue
        }
        if height <= afterHeight {
            continue
        }
        // Unmarshal value into types.Data
        d := new(types.Data)
        if err := d.UnmarshalBinary(res.Value); err != nil {
            return nil, fmt.Errorf("failed to unmarshal data at key %q: %w", res.Key, err)
        }
        items = append(items, pair{h: height, d: d})
    }

    // Ensure result is sorted by numeric height
    sort.Slice(items, func(i, j int) bool { return items[i].h < items[j].h })

    out := make([]*types.Data, 0, len(items))
    for _, it := range items {
        out = append(out, it.d)
    }
    return out, nil
}

// HeadersAfterHeight returns all SignedHeader items with height greater than
// the provided height, sorted by height ascending.
func (s *DefaultStore) HeadersAfterHeight(ctx context.Context, afterHeight uint64) ([]*types.SignedHeader, error) {
    prefix := GenerateKey([]string{headerPrefix}) // "/h"
    results, err := s.db.Query(ctx, dsq.Query{Prefix: prefix})
    if err != nil {
        return nil, fmt.Errorf("datastore query failed: %w", err)
    }
    defer results.Close()

    type pair struct {
        h uint64
        hd *types.SignedHeader
    }
    items := make([]pair, 0, 1024)

    keyPrefix := prefix + "/"
    for res := range results.Next() {
        if res.Error != nil {
            return nil, fmt.Errorf("datastore iteration error: %w", res.Error)
        }
        if !strings.HasPrefix(res.Key, keyPrefix) {
            continue
        }
        hs := strings.TrimPrefix(res.Key, keyPrefix)
        height, err := strconv.ParseUint(hs, 10, 64)
        if err != nil {
            continue
        }
        if height <= afterHeight {
            continue
        }
        h := new(types.SignedHeader)
        if err := h.UnmarshalBinary(res.Value); err != nil {
            return nil, fmt.Errorf("failed to unmarshal header at key %q: %w", res.Key, err)
        }
        items = append(items, pair{h: height, hd: h})
    }

    sort.Slice(items, func(i, j int) bool { return items[i].h < items[j].h })

    out := make([]*types.SignedHeader, 0, len(items))
    for _, it := range items {
        out = append(out, it.hd)
    }
    return out, nil
}
