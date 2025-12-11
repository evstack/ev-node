package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
	"github.com/stretchr/testify/assert"
)

func TestExchangeWrapper_Get(t *testing.T) {
	ctx := context.Background()
	hash := header.Hash([]byte("test-hash"))
	expectedHeader := &types.SignedHeader{} // Just a dummy

	t.Run("Hit in Store", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		// Exchange should NOT be called

		getter := func(ctx context.Context, s store.Store, h header.Hash) (*types.SignedHeader, error) {
			return expectedHeader, nil
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			daStore:     mocks.NewMockStore(t),
			getter:      getter,
		}

		h, err := ew.Get(ctx, hash)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})

	t.Run("Miss in Store", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		mockEx.EXPECT().Get(ctx, hash).Return(expectedHeader, nil)

		getter := func(ctx context.Context, s store.Store, h header.Hash) (*types.SignedHeader, error) {
			return nil, errors.New("not found")
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			daStore:     mocks.NewMockStore(t),
			getter:      getter,
		}

		h, err := ew.Get(ctx, hash)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})

	t.Run("Store Not Configured", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		mockEx.EXPECT().Get(ctx, hash).Return(expectedHeader, nil)

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			daStore:     nil, // No store
			getter:      nil,
		}

		h, err := ew.Get(ctx, hash)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})
}

func TestExchangeWrapper_GetByHeight(t *testing.T) {
	ctx := context.Background()
	height := uint64(10)
	expectedHeader := &types.SignedHeader{}

	t.Run("Hit in Store", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)

		getterByHeight := func(ctx context.Context, s store.Store, h uint64) (*types.SignedHeader, error) {
			return expectedHeader, nil
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange:    mockEx,
			daStore:        mocks.NewMockStore(t),
			getterByHeight: getterByHeight,
		}

		h, err := ew.GetByHeight(ctx, height)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})

	t.Run("Miss in Store", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		mockEx.EXPECT().GetByHeight(ctx, height).Return(expectedHeader, nil)

		getterByHeight := func(ctx context.Context, s store.Store, h uint64) (*types.SignedHeader, error) {
			return nil, errors.New("not found")
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange:    mockEx,
			daStore:        mocks.NewMockStore(t),
			getterByHeight: getterByHeight,
		}

		h, err := ew.GetByHeight(ctx, height)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})
}
