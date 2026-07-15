package evm

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestBuildPayloadAttributes_AddsSlotNumberCandidate(t *testing.T) {
	client := &EngineClient{}

	attrs := client.buildPayloadAttributes(nil, 12, time.Unix(1_700_000_000, 0), 30_000_000)
	require.Equal(t, hexutil.Uint64(12), attrs["slotNumber"])
}
