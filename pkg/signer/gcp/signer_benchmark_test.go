package gcp

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// BenchmarkKmsSignerSign test kms round trip
// export EVNODE_E2E_AWS_KMS_KEY_ID="arn:aws:kms:...."
// export EVNODE_E2E_AWS_KMS_REGION=eu-west-1
// go test  -v -bench=BenchmarkKmsSignerSign -benchtime=3s -count=10 -run='^$' ./pkg/signer/aws
func BenchmarkKmsSignerSign(b *testing.B) {
	keyName := os.Getenv("EVNODE_E2E_GCP_KMS_KEY_NAME")
	if keyName == "" {
		b.Skip("set EVNODE_E2E_GCP_KMS_KEY_NAME to run GCP KMS benchmark")
	}

	credentialsFile := os.Getenv("EVNODE_E2E_GCP_KMS_CREDENTIALS_FILE")

	signer, err := NewKmsSigner(b.Context(), keyName, &Options{
		CredentialsFile: credentialsFile,
		Timeout:         1 * time.Second,
		MaxRetries:      0,
	})
	require.NoError(b, err)

	headerData := benchmarkHeaderData(b)

	b.ReportAllocs()
	b.SetBytes(int64(len(headerData)))
	b.ResetTimer()

	for b.Loop() {
		_, err := signer.Sign(b.Context(), headerData)
		require.NoError(b, err)
	}
}

func benchmarkHeaderData(b *testing.B) []byte {
	b.Helper()

	header := &pb.Header{
		Version: &pb.Version{
			Block: 11,
			App:   1,
		},
		Height:          1,
		Time:            1743091200000000000, // 2025-03-27T00:00:00Z
		LastHeaderHash:  bytes.Repeat([]byte{0x3b}, 32),
		DataHash:        bytes.Repeat([]byte{0x78}, 32),
		AppHash:         bytes.Repeat([]byte{0xf5}, 32),
		ProposerAddress: bytes.Repeat([]byte{0x25}, 32),
		ValidatorHash:   bytes.Repeat([]byte{0x25}, 32),
		ChainId:         "benchmark-chain",
	}

	headerData, err := proto.Marshal(header)
	require.NoError(b, err)
	require.NotEmpty(b, headerData)

	return headerData
}
