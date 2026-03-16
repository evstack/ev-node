package aws

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKMSClient is a test double implementing KMSClient.
type mockKMSClient struct {
	pubKeyDER []byte
	signFn    func(ctx context.Context, params *kms.SignInput) (*kms.SignOutput, error)
	getPubFn  func(ctx context.Context, params *kms.GetPublicKeyInput) (*kms.GetPublicKeyOutput, error)
}

func (m *mockKMSClient) Sign(ctx context.Context, params *kms.SignInput, _ ...func(*kms.Options)) (*kms.SignOutput, error) {
	if m.signFn != nil {
		return m.signFn(ctx, params)
	}
	return &kms.SignOutput{Signature: []byte("mock-signature")}, nil
}

func (m *mockKMSClient) GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, _ ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	if m.getPubFn != nil {
		return m.getPubFn(ctx, params)
	}
	return &kms.GetPublicKeyOutput{
		PublicKey: m.pubKeyDER,
	}, nil
}

// generateTestEd25519DER generates an Ed25519 key pair and returns
// the public key in DER (X.509 SubjectPublicKeyInfo) format.
func generateTestEd25519DER(t *testing.T) (ed25519.PublicKey, []byte) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	return pub, der
}

func TestNewKmsSignerFromClient_Success(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{pubKeyDER: der}
	s, err := NewKmsSignerFromClient(context.Background(), mock, "arn:aws:kms:us-east-1:123456789012:key/test-key-id", nil)
	require.NoError(t, err)
	require.NotNil(t, s)

	// Verify public key was cached
	pubKey, err := s.GetPublic()
	require.NoError(t, err)
	require.NotNil(t, pubKey)

	// Verify address was cached
	addr, err := s.GetAddress()
	require.NoError(t, err)
	assert.Len(t, addr, 32) // sha256 output
}

func TestNewKmsSignerFromClient_EmptyKeyID(t *testing.T) {
	_, err := NewKmsSignerFromClient(context.Background(), &mockKMSClient{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key ID is required")
}

func TestNewKmsSignerFromClient_GetPublicKeyFails(t *testing.T) {
	mock := &mockKMSClient{
		getPubFn: func(_ context.Context, _ *kms.GetPublicKeyInput) (*kms.GetPublicKeyOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	_, err := NewKmsSignerFromClient(context.Background(), mock, "test-key", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestSign_Success(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	expectedSig := []byte("test-signature-bytes")
	mock := &mockKMSClient{
		pubKeyDER: der,
		signFn: func(_ context.Context, params *kms.SignInput) (*kms.SignOutput, error) {
			assert.Equal(t, types.MessageTypeRaw, params.MessageType)
			assert.Equal(t, types.SigningAlgorithmSpecEd25519Sha512, params.SigningAlgorithm)
			return &kms.SignOutput{Signature: expectedSig}, nil
		},
	}

	s, err := NewKmsSignerFromClient(context.Background(), mock, "test-key", nil)
	require.NoError(t, err)

	sig, err := s.Sign(context.Background(), []byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, expectedSig, sig)
}

func TestSign_KMSFailure(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{
		pubKeyDER: der,
		signFn: func(_ context.Context, _ *kms.SignInput) (*kms.SignOutput, error) {
			return nil, fmt.Errorf("throttling exception")
		},
	}

	s, err := NewKmsSignerFromClient(context.Background(), mock, "test-key", nil)
	require.NoError(t, err)

	_, err = s.Sign(context.Background(), []byte("hello world"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "KMS Sign failed")
}

func TestGetPublic_Cached(t *testing.T) {
	pub, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{pubKeyDER: der}
	s, err := NewKmsSignerFromClient(context.Background(), mock, "test-key", nil)
	require.NoError(t, err)

	cryptoPub, err := s.GetPublic()
	require.NoError(t, err)

	raw, err := cryptoPub.Raw()
	require.NoError(t, err)
	assert.Equal(t, []byte(pub), raw)
}

func TestGetAddress_Deterministic(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{pubKeyDER: der}
	s, err := NewKmsSignerFromClient(context.Background(), mock, "test-key", nil)
	require.NoError(t, err)

	addr1, err := s.GetAddress()
	require.NoError(t, err)

	addr2, err := s.GetAddress()
	require.NoError(t, err)

	assert.Equal(t, addr1, addr2, "address should be deterministic")
}
