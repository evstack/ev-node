package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var dataHashForEmptyTxs = []byte{110, 52, 11, 156, 255, 179, 122, 152, 156, 165, 68, 230, 187, 120, 10, 44, 120, 144, 29, 63, 179, 55, 56, 118, 133, 17, 163, 6, 23, 175, 160, 29}

func TestAssertValidForNextState(t *testing.T) {
	// Define test table
	now := time.Now()
	nowUnixNano := uint64(now.UnixNano())
	testCases := map[string]struct {
		state         State
		header        *SignedHeader
		data          *Data
		expectedError string
	}{
		"valid initial state": {
			state: State{
				ChainID: "test-chain",
			},
			header: &SignedHeader{
				Header: Header{
					BaseHeader: BaseHeader{
						ChainID: "test-chain", Height: 1,
					},
					DataHash: dataHashForEmptyTxs,
				},
			},
			data:          &Data{},
			expectedError: "",
		},
		"chain ID mismatch": {
			state: State{
				ChainID: "test-chain",
			},
			header: &SignedHeader{
				Header: Header{
					BaseHeader: BaseHeader{
						ChainID: "wrong-chain", Height: 1,
					},
					DataHash: dataHashForEmptyTxs,
				},
			},
			data:          &Data{},
			expectedError: "invalid chain ID",
		},
		"invalid block height": {
			state: State{
				ChainID:         "test-chain",
				LastHeaderHash:  []byte("hash"),
				LastBlockTime:   now,
				LastBlockHeight: 5,
			},
			header: &SignedHeader{
				Header: Header{
					BaseHeader: BaseHeader{
						ChainID: "test-chain", Height: 7,
						Time: nowUnixNano + 1,
					},
					DataHash:       dataHashForEmptyTxs,
					LastHeaderHash: []byte("hash"),
				},
			},
			data:          &Data{},
			expectedError: "invalid block height",
		},
		"invalid block time": {
			state: State{
				ChainID:         "test-chain",
				LastHeaderHash:  []byte("hash"),
				LastBlockHeight: 1,
				LastBlockTime:   now,
			},
			header: &SignedHeader{
				Header: Header{
					BaseHeader: BaseHeader{
						ChainID: "test-chain",
						Height:  2,
						Time:    nowUnixNano - 1,
					},
					DataHash:       dataHashForEmptyTxs,
					LastHeaderHash: []byte("hash"),
				},
			},
			data:          &Data{},
			expectedError: "invalid block time",
		},
		"invalid data hash": {
			state: State{
				ChainID:         "test-chain",
				LastHeaderHash:  []byte("hash"),
				LastBlockHeight: 1,
				LastBlockTime:   now,
			},
			header: &SignedHeader{
				Header: Header{
					BaseHeader: BaseHeader{
						ChainID: "test-chain",
						Height:  2,
						Time:    nowUnixNano,
					},
					DataHash:       []byte("other-hash"),
					LastHeaderHash: []byte("hash"),
				},
			},
			data:          &Data{},
			expectedError: "dataHash from the header does not match with hash",
		},
		"last header hash mismatch": {
			state: State{
				ChainID:         "test-chain",
				LastHeaderHash:  []byte("hash"),
				LastBlockHeight: 1,
				LastBlockTime:   now,
			},
			header: &SignedHeader{
				Header: Header{
					BaseHeader: BaseHeader{
						ChainID: "test-chain", Height: 2,
						Time: nowUnixNano,
					},
					DataHash:       dataHashForEmptyTxs,
					LastHeaderHash: []byte("other-hash"),
				},
			},
			data:          &Data{},
			expectedError: "invalid last header hash",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.state.AssertValidForNextState(tc.header, tc.data)
			if tc.expectedError == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.expectedError)
		})
	}
}
