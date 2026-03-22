package evm

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core/types"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestReconcileExecutionAtHeight_StartedExecMeta(t *testing.T) {
	t.Parallel()

	specs := map[string]struct {
		execMetaTimestamp int64
		execMetaTxs       [][]byte
		requestedTxs      [][]byte
		requestedTime     time.Time
		expectFound       bool
		expectPayloadID   bool
		expectGetPayloads int
	}{
		"resume_when_inputs_match": {
			execMetaTimestamp: 1700000012,
			execMetaTxs:       [][]byte{[]byte("tx-1")},
			requestedTxs:      [][]byte{[]byte("tx-1")},
			requestedTime:     time.Unix(1700000012, 0),
			expectFound:       true,
			expectPayloadID:   true,
			expectGetPayloads: 1,
		},
		"ignore_when_timestamp_differs": {
			execMetaTimestamp: 1700000010,
			execMetaTxs:       [][]byte{[]byte("tx-1")},
			requestedTxs:      [][]byte{[]byte("tx-1")},
			requestedTime:     time.Unix(1700000012, 0),
			expectFound:       false,
			expectPayloadID:   false,
			expectGetPayloads: 0,
		},
		"ignore_when_txs_differ": {
			execMetaTimestamp: 1700000012,
			execMetaTxs:       [][]byte{[]byte("tx-old")},
			requestedTxs:      [][]byte{[]byte("tx-new")},
			requestedTime:     time.Unix(1700000012, 0),
			expectFound:       false,
			expectPayloadID:   false,
			expectGetPayloads: 0,
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store := NewEVMStore(dssync.MutexWrap(ds.NewMapDatastore()))
			payloadID := engine.PayloadID{1, 2, 3, 4, 5, 6, 7, 8}
			require.NoError(t, store.SaveExecMeta(t.Context(), &ExecMeta{
				Height:    12,
				PayloadID: payloadID[:],
				TxHash:    hashTxs(spec.execMetaTxs),
				Timestamp: spec.execMetaTimestamp,
				Stage:     ExecStageStarted,
			}))

			engineRPC := &mockReconcileEngineRPCClient{
				payloads: map[engine.PayloadID]*engine.ExecutionPayloadEnvelope{
					payloadID: {},
				},
			}
			client := &EngineClient{
				engineClient: engineRPC,
				ethClient:    mockReconcileEthRPCClient{},
				store:        store,
				logger:       zerolog.Nop(),
			}

			stateRoot, gotPayloadID, found, err := client.reconcileExecutionAtHeight(t.Context(), 12, spec.requestedTime, spec.requestedTxs)

			require.NoError(t, err)
			require.Nil(t, stateRoot)
			require.Equal(t, spec.expectFound, found)
			require.Equal(t, spec.expectPayloadID, gotPayloadID != nil)
			if spec.expectPayloadID {
				require.Equal(t, payloadID, *gotPayloadID)
			}
			require.Equal(t, spec.expectGetPayloads, engineRPC.getPayloadCalls)
		})
	}
}

type mockReconcileEngineRPCClient struct {
	payloads        map[engine.PayloadID]*engine.ExecutionPayloadEnvelope
	getPayloadCalls int
}

func (m *mockReconcileEngineRPCClient) ForkchoiceUpdated(_ context.Context, _ engine.ForkchoiceStateV1, _ map[string]any) (*engine.ForkChoiceResponse, error) {
	return nil, errors.New("unexpected ForkchoiceUpdated call")
}

func (m *mockReconcileEngineRPCClient) GetPayload(_ context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	m.getPayloadCalls++
	payload, ok := m.payloads[payloadID]
	if !ok {
		return nil, errors.New("payload not found")
	}

	return payload, nil
}

func (m *mockReconcileEngineRPCClient) NewPayload(_ context.Context, _ *engine.ExecutableData, _ []string, _ string, _ [][]byte) (*engine.PayloadStatusV1, error) {
	return nil, errors.New("unexpected NewPayload call")
}

type mockReconcileEthRPCClient struct{}

func (mockReconcileEthRPCClient) HeaderByNumber(_ context.Context, _ *big.Int) (*types.Header, error) {
	return nil, errors.New("header not found")
}

func (mockReconcileEthRPCClient) GetTxs(_ context.Context) ([]string, error) {
	return nil, errors.New("unexpected GetTxs call")
}
