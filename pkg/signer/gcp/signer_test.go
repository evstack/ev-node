package gcp

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync/atomic"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	mock := &mockKMSClient{
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			assert.Equal(t, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", req.Name)
			assert.Equal(t, []byte("hello world"), req.Data)
			return &kmspb.AsymmetricSignResponse{Signature: expectedSig}, nil
		},
	}

	s, err := kmsSignerFromClient(context.Background(), mock, "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1", nil)
	require.NoError(t, err)

	sig, err := s.Sign(context.Background(), []byte("hello world"))
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
