package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/celestiaorg/go-header"
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

		getter := func(ctx context.Context, h header.Hash) (*types.SignedHeader, error) {
			return expectedHeader, nil
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			getter:      getter,
		}

		h, err := ew.Get(ctx, hash)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})

	t.Run("Miss in Store", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		mockEx.EXPECT().Get(ctx, hash).Return(expectedHeader, nil)

		getter := func(ctx context.Context, h header.Hash) (*types.SignedHeader, error) {
			return nil, errors.New("not found")
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			getter:      getter,
		}

		h, err := ew.Get(ctx, hash)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})

	t.Run("Getter Not Configured", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		mockEx.EXPECT().Get(ctx, hash).Return(expectedHeader, nil)

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
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

		getterByHeight := func(ctx context.Context, h uint64) (*types.SignedHeader, error) {
			return expectedHeader, nil
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange:    mockEx,
			getterByHeight: getterByHeight,
		}

		h, err := ew.GetByHeight(ctx, height)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})

	t.Run("Miss in Store", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)
		mockEx.EXPECT().GetByHeight(ctx, height).Return(expectedHeader, nil)

		getterByHeight := func(ctx context.Context, h uint64) (*types.SignedHeader, error) {
			return nil, errors.New("not found")
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange:    mockEx,
			getterByHeight: getterByHeight,
		}

		h, err := ew.GetByHeight(ctx, height)
		assert.NoError(t, err)
		assert.Equal(t, expectedHeader, h)
	})
}

func TestExchangeWrapper_GetRangeByHeight(t *testing.T) {
	ctx := context.Background()

	t.Run("All from DA", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)

		headers := []*types.SignedHeader{
			{Header: types.Header{BaseHeader: types.BaseHeader{Height: 2}}},
			{Header: types.Header{BaseHeader: types.BaseHeader{Height: 3}}},
		}
		from := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 1}}}

		rangeGetter := func(ctx context.Context, fromH, toH uint64) ([]*types.SignedHeader, uint64, error) {
			return headers, 4, nil
		}

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			rangeGetter: rangeGetter,
		}

		result, err := ew.GetRangeByHeight(ctx, from, 4)
		assert.NoError(t, err)
		assert.Equal(t, headers, result)
	})

	t.Run("Partial from DA then P2P", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)

		daHeaders := []*types.SignedHeader{
			{Header: types.Header{BaseHeader: types.BaseHeader{Height: 2}}},
		}
		p2pHeaders := []*types.SignedHeader{
			{Header: types.Header{BaseHeader: types.BaseHeader{Height: 3}}},
		}
		from := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 1}}}

		rangeGetter := func(ctx context.Context, fromH, toH uint64) ([]*types.SignedHeader, uint64, error) {
			return daHeaders, 3, nil // only got height 2, need 3+
		}
		mockEx.EXPECT().GetRangeByHeight(ctx, daHeaders[0], uint64(4)).Return(p2pHeaders, nil)

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			rangeGetter: rangeGetter,
		}

		result, err := ew.GetRangeByHeight(ctx, from, 4)
		assert.NoError(t, err)
		assert.Equal(t, append(daHeaders, p2pHeaders...), result)
	})

	t.Run("No range getter fallback to P2P", func(t *testing.T) {
		mockEx := extmocks.NewMockP2PExchange[*types.SignedHeader](t)

		from := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: 1}}}
		expected := []*types.SignedHeader{
			{Header: types.Header{BaseHeader: types.BaseHeader{Height: 2}}},
		}
		mockEx.EXPECT().GetRangeByHeight(ctx, from, uint64(3)).Return(expected, nil)

		ew := &exchangeWrapper[*types.SignedHeader]{
			p2pExchange: mockEx,
			rangeGetter: nil,
		}

		result, err := ew.GetRangeByHeight(ctx, from, 3)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}
