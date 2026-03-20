package gcp

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash/crc32"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// mockKMSClient is a test double implementing KMSClient.
type mockKMSClient struct {
	publicKeyPEM string
	signFn       func(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error)
	getPubFn     func(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error)
}

func (m *mockKMSClient) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	if m.signFn != nil {
		return m.signFn(ctx, req)
	}
	return &kmspb.AsymmetricSignResponse{Signature: []byte("mock-signature")}, nil
}

func (m *mockKMSClient) GetPublicKey(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
	if m.getPubFn != nil {
		return m.getPubFn(ctx, req)
	}
	return &kmspb.PublicKey{Pem: m.publicKeyPEM}, nil
}

// generateTestEd25519PEM generates an Ed25519 key pair and returns
// the public key in PEM format.
func generateTestEd25519PEM(t *testing.T) (ed25519.PublicKey, string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return pub, string(pemBytes)
}

func TestNewKmsSignerFromClient_Success(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	mock := &mockKMSClient{publicKeyPEM: publicKeyPEM}
	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.NoError(t, err)
	require.NotNil(t, s)

	// Verify public key was cached.
	pubKey, err := s.GetPublic()
	require.NoError(t, err)
	require.NotNil(t, pubKey)

	// Verify address was cached.
	addr, err := s.GetAddress()
	require.NoError(t, err)
	assert.Len(t, addr, 32) // sha256 output
}

func TestNewKmsSignerFromClient_EmptyKeyName(t *testing.T) {
	_, err := kmsSignerFromClient(context.Background(), &mockKMSClient{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key name is required")
}

func TestNewKmsSignerFromClient_NilClient(t *testing.T) {
	_, err := kmsSignerFromClient(context.Background(), nil, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestNewKmsSignerFromClient_GetPublicKeyFails(t *testing.T) {
	mock := &mockKMSClient{
		getPubFn: func(_ context.Context, _ *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
			return nil, fmt.Errorf("permission denied")
		},
	}

	_, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestSign_Success(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	expectedSig := []byte("test-signature-bytes")
	expectedMsg := []byte("hello world")
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			assert.Equal(t, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", req.Name)
			assert.Equal(t, expectedMsg, req.Data)
			require.NotNil(t, req.DataCrc32C)
			assert.Equal(t, int64(crc32.Checksum(expectedMsg, castagnoliTable)), req.DataCrc32C.GetValue())
			return &kmspb.AsymmetricSignResponse{
				Signature:          expectedSig,
				VerifiedDataCrc32C: true,
				SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(expectedSig, castagnoliTable))),
			}, nil
		},
	}

	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.NoError(t, err)

	sig, err := s.Sign(context.Background(), expectedMsg)
	require.NoError(t, err)
	assert.Equal(t, expectedSig, sig)
}

func TestSign_KMSFailure(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls int32
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			atomic.AddInt32(&calls, 1)
			return nil, status.Error(codes.Unavailable, "temporarily unavailable")
		},
	}

	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.NoError(t, err)

	_, err = s.Sign(context.Background(), []byte("hello world"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "KMS Sign failed")
	assert.Equal(t, int32(4), atomic.LoadInt32(&calls), "default retries should make 4 attempts")
}

func TestSign_MaxRetriesZero_DisablesRetries(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls int32
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			atomic.AddInt32(&calls, 1)
			return nil, status.Error(codes.Unavailable, "temporarily unavailable")
		},
	}

	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", &Options{MaxRetries: 0})
	require.NoError(t, err)

	_, err = s.Sign(context.Background(), []byte("hello world"))
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "max retries 0 should only make one attempt")
}

func TestSign_NonRetryableError_NoRetries(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls int32
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			atomic.AddInt32(&calls, 1)
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		},
	}

	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", &Options{MaxRetries: 3})
	require.NoError(t, err)

	_, err = s.Sign(context.Background(), []byte("hello world"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-retryable")
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "non-retryable errors should fail fast")
}

func TestSign_IntegrityCheckVerifiedDataFalse_RetriesAndFails(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls int32
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			atomic.AddInt32(&calls, 1)
			sig := []byte("sig")
			return &kmspb.AsymmetricSignResponse{
				Signature:          sig,
				VerifiedDataCrc32C: false,
				SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(sig, castagnoliTable))),
			}, nil
		},
	}

	s, err := kmsSignerFromClient(
		context.Background(),
		mock,
		"projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1",
		&Options{MaxRetries: 1},
	)
	require.NoError(t, err)

	_, err = s.Sign(context.Background(), []byte("hello world"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verified_data_crc32c is false")
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "integrity failures should be retried")
}

func TestSign_IntegrityCheckSignatureCRC32CMismatch_RetriesAndFails(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls int32
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			atomic.AddInt32(&calls, 1)
			return &kmspb.AsymmetricSignResponse{
				Signature:          []byte("sig"),
				VerifiedDataCrc32C: true,
				SignatureCrc32C:    wrapperspb.Int64(12345),
			}, nil
		},
	}

	s, err := kmsSignerFromClient(
		context.Background(),
		mock,
		"projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1",
		&Options{MaxRetries: 1},
	)
	require.NoError(t, err)

	_, err = s.Sign(context.Background(), []byte("hello world"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature_crc32c mismatch")
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "integrity failures should be retried")
}

func TestSign_IntegrityCheckRecoversOnRetry(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls int32
	expectedSig := []byte("valid-signature")
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			attempt := atomic.AddInt32(&calls, 1)
			if attempt == 1 {
				return &kmspb.AsymmetricSignResponse{
					Signature:          []byte("corrupted"),
					VerifiedDataCrc32C: false,
					SignatureCrc32C:    wrapperspb.Int64(1),
				}, nil
			}
			return &kmspb.AsymmetricSignResponse{
				Signature:          expectedSig,
				VerifiedDataCrc32C: true,
				SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(expectedSig, castagnoliTable))),
			}, nil
		},
	}

	s, err := kmsSignerFromClient(
		context.Background(),
		mock,
		"projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1",
		&Options{MaxRetries: 2},
	)
	require.NoError(t, err)

	got, err := s.Sign(context.Background(), []byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, expectedSig, got)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "second attempt should succeed")
}

func TestRetryBackoff_Capped(t *testing.T) {
	testCases := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{name: "attempt 1", attempt: 1, expected: 100 * time.Millisecond},
		{name: "attempt 2", attempt: 2, expected: 200 * time.Millisecond},
		{name: "attempt 6", attempt: 6, expected: 3200 * time.Millisecond},
		{name: "attempt 7 capped", attempt: 7, expected: 5 * time.Second},
		{name: "attempt 10 capped", attempt: 10, expected: 5 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, retryBackoff(tc.attempt))
		})
	}
}

func TestGetPublic_Cached(t *testing.T) {
	pub, publicKeyPEM := generateTestEd25519PEM(t)

	mock := &mockKMSClient{publicKeyPEM: publicKeyPEM}
	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.NoError(t, err)

	cryptoPub, err := s.GetPublic()
	require.NoError(t, err)

	raw, err := cryptoPub.Raw()
	require.NoError(t, err)
	assert.Equal(t, []byte(pub), raw)
}

func TestGetAddress_Deterministic(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	mock := &mockKMSClient{publicKeyPEM: publicKeyPEM}
	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.NoError(t, err)

	addr1, err := s.GetAddress()
	require.NoError(t, err)

	addr2, err := s.GetAddress()
	require.NoError(t, err)

	assert.Equal(t, addr1, addr2, "address should be deterministic")
}
