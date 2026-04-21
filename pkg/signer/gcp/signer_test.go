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

const myKeyID = "projects/p/locations/global/keyRings/r/cryptoKeys/k/cryptoKeyVersions/1"

// mockKMSClient is a test double implementing KMSClient.
type mockKMSClient struct {
	publicKeyPEM string
	signFn       func(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error)
	getPubFn     func(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error)
	keyID        string
}

func (m *mockKMSClient) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	if m.signFn != nil {
		return m.signFn(ctx, req)
	}
	return &kmspb.AsymmetricSignResponse{Signature: []byte("mock-signature"), Name: m.keyID}, nil
}

func (m *mockKMSClient) GetPublicKey(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
	if m.getPubFn != nil {
		return m.getPubFn(ctx, req)
	}
	crc := crc32.Checksum([]byte(m.publicKeyPEM), castagnoliTable)
	return &kmspb.PublicKey{Pem: m.publicKeyPEM, Name: m.keyID, PemCrc32C: wrapperspb.Int64(int64(crc))}, nil
}

func TestNewKmsSignerFromClient_Success(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	mock := &mockKMSClient{publicKeyPEM: publicKeyPEM, keyID: myKeyID}
	s, err := kmsSignerFromClient(t.Context(), mock, myKeyID, nil)
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

func TestNewKmsSignerFromClient_Validation(t *testing.T) {
	specs := map[string]struct {
		client    KMSClient
		keyName   string
		errSubstr string
	}{
		"empty key name": {
			client:    &mockKMSClient{},
			keyName:   "",
			errSubstr: "key name is required",
		},
		"nil client": {
			client:    nil,
			keyName:   myKeyID,
			errSubstr: "client is required",
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			_, err := kmsSignerFromClient(t.Context(), spec.client, spec.keyName, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), spec.errSubstr)
		})
	}
}

func TestNewKmsSignerFromClient_GetPublicKeyFails(t *testing.T) {
	mock := &mockKMSClient{
		keyID: myKeyID,
		getPubFn: func(_ context.Context, _ *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
			return nil, fmt.Errorf("permission denied")
		},
	}

	_, err := kmsSignerFromClient(t.Context(), mock, myKeyID, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestSign_Success(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	expectedSig := []byte("test-signature-bytes")
	expectedMsg := []byte("hello world")
	mock := &mockKMSClient{
		keyID:        myKeyID,
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			assert.Equal(t, myKeyID, req.Name)
			assert.Equal(t, expectedMsg, req.Data)
			require.NotNil(t, req.DataCrc32C)
			assert.Equal(t, int64(crc32.Checksum(expectedMsg, castagnoliTable)), req.DataCrc32C.GetValue())
			return &kmspb.AsymmetricSignResponse{
				Name:               myKeyID,
				Signature:          expectedSig,
				VerifiedDataCrc32C: true,
				SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(expectedSig, castagnoliTable))),
			}, nil
		},
	}

	s, err := kmsSignerFromClient(t.Context(), mock, myKeyID, nil)
	require.NoError(t, err)

	sig, err := s.Sign(t.Context(), expectedMsg)
	require.NoError(t, err)
	assert.Equal(t, expectedSig, sig)
}

func TestSign_RetryBehavior(t *testing.T) {
	specs := map[string]struct {
		opts         *Options
		signErr      error
		errSubstr    string
		expectedCall int32
	}{
		"retryable uses default retries": {
			opts:         nil,
			signErr:      status.Error(codes.Unavailable, "temporarily unavailable"),
			errSubstr:    "KMS Sign failed",
			expectedCall: 4,
		},
		"max retries zero disables retries": {
			opts:         &Options{MaxRetries: 0},
			signErr:      status.Error(codes.Unavailable, "temporarily unavailable"),
			errSubstr:    "KMS Sign failed",
			expectedCall: 1,
		},
		"non retryable fails fast": {
			opts:         &Options{MaxRetries: 3},
			signErr:      status.Error(codes.PermissionDenied, "permission denied"),
			errSubstr:    "non-retryable",
			expectedCall: 1,
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			_, publicKeyPEM := generateTestEd25519PEM(t)

			var calls atomic.Int32
			signer := newTestSigner(t, &mockKMSClient{
				keyID:        myKeyID,
				publicKeyPEM: publicKeyPEM,
				signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
					calls.Add(1)
					return nil, spec.signErr
				},
			}, spec.opts)

			_, err := signer.Sign(t.Context(), []byte("hello world"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), spec.errSubstr)
			assert.Equal(t, spec.expectedCall, calls.Load())
		})
	}
}

func TestSign_IntegrityFailures_RetryAndFail(t *testing.T) {
	specs := map[string]struct {
		expectedErrSubstr string
		responseFn        func() *kmspb.AsymmetricSignResponse
	}{
		"verified data false": {
			expectedErrSubstr: "verified_data_crc32c is false",
			responseFn: func() *kmspb.AsymmetricSignResponse {
				sig := []byte("sig")
				return &kmspb.AsymmetricSignResponse{
					Name:               myKeyID,
					Signature:          sig,
					VerifiedDataCrc32C: false,
					SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(sig, castagnoliTable))),
				}
			},
		},
		"signature crc mismatch": {
			expectedErrSubstr: "signature_crc32c mismatch",
			responseFn: func() *kmspb.AsymmetricSignResponse {
				return &kmspb.AsymmetricSignResponse{
					Name:               myKeyID,
					Signature:          []byte("sig"),
					VerifiedDataCrc32C: true,
					SignatureCrc32C:    wrapperspb.Int64(12345),
				}
			},
		},
		"key name mismatch": {
			expectedErrSubstr: "unexpected key name",
			responseFn: func() *kmspb.AsymmetricSignResponse {
				sig := []byte("sig")
				return &kmspb.AsymmetricSignResponse{
					Name:               "projects/p/locations/global/keyRings/r/cryptoKeys/other/cryptoKeyVersions/1",
					Signature:          sig,
					VerifiedDataCrc32C: true,
					SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(sig, castagnoliTable))),
				}
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			_, publicKeyPEM := generateTestEd25519PEM(t)

			var calls atomic.Int32
			signer := newTestSigner(t, &mockKMSClient{
				keyID:        myKeyID,
				publicKeyPEM: publicKeyPEM,
				signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
					calls.Add(1)
					return spec.responseFn(), nil
				},
			}, &Options{MaxRetries: 1})

			_, err := signer.Sign(t.Context(), []byte("hello world"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), spec.expectedErrSubstr)
			assert.Equal(t, int32(2), calls.Load(), "integrity failures should be retried")
		})
	}
}

func TestSign_IntegrityCheckRecoversOnRetry(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	var calls atomic.Int32
	expectedSig := []byte("valid-signature")
	mock := &mockKMSClient{
		keyID:        myKeyID,
		publicKeyPEM: publicKeyPEM,
		signFn: func(_ context.Context, _ *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
			attempt := calls.Add(1)
			if attempt == 1 {
				return &kmspb.AsymmetricSignResponse{
					Name:               myKeyID,
					Signature:          []byte("corrupted"),
					VerifiedDataCrc32C: false,
					SignatureCrc32C:    wrapperspb.Int64(1),
				}, nil
			}
			return &kmspb.AsymmetricSignResponse{
				Name:               myKeyID,
				Signature:          expectedSig,
				VerifiedDataCrc32C: true,
				SignatureCrc32C:    wrapperspb.Int64(int64(crc32.Checksum(expectedSig, castagnoliTable))),
			}, nil
		},
	}

	s := newTestSigner(t, mock, &Options{MaxRetries: 2})

	got, err := s.Sign(t.Context(), []byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, expectedSig, got)
	assert.Equal(t, int32(2), calls.Load(), "second attempt should succeed")
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

	mock := &mockKMSClient{publicKeyPEM: publicKeyPEM, keyID: myKeyID}
	s := newTestSigner(t, mock, nil)

	cryptoPub, err := s.GetPublic()
	require.NoError(t, err)

	raw, err := cryptoPub.Raw()
	require.NoError(t, err)
	assert.Equal(t, []byte(pub), raw)
}

func TestGetAddress_Deterministic(t *testing.T) {
	_, publicKeyPEM := generateTestEd25519PEM(t)

	mock := &mockKMSClient{publicKeyPEM: publicKeyPEM, keyID: myKeyID}
	s := newTestSigner(t, mock, nil)

	addr1, err := s.GetAddress()
	require.NoError(t, err)

	addr2, err := s.GetAddress()
	require.NoError(t, err)

	assert.Equal(t, addr1, addr2, "address should be deterministic")
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

func newTestSigner(t *testing.T, mock *mockKMSClient, opts *Options) *KmsSigner {
	t.Helper()
	s, err := kmsSignerFromClient(t.Context(), mock, myKeyID, opts)
	require.NoError(t, err)
	return s
}
