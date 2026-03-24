package aws

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awsTestKeyID = "arn:aws:kms:us-east-1:123456789012:key/test-key-id"

// mockKMSClient is a test double implementing KMSClient.
type mockKMSClient struct {
	pubKeyDER []byte
	signFn    func(ctx context.Context, params *kms.SignInput) (*kms.SignOutput, error)
	getPubFn  func(ctx context.Context, params *kms.GetPublicKeyInput) (*kms.GetPublicKeyOutput, error)
	keyID     string
}

func (m *mockKMSClient) Sign(ctx context.Context, params *kms.SignInput, _ ...func(*kms.Options)) (*kms.SignOutput, error) {
	if m.signFn != nil {
		return m.signFn(ctx, params)
	}
	return &kms.SignOutput{Signature: []byte("mock-signature"), KeyId: &m.keyID}, nil
}

func (m *mockKMSClient) GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, _ ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	if m.getPubFn != nil {
		return m.getPubFn(ctx, params)
	}
	return &kms.GetPublicKeyOutput{
		PublicKey: m.pubKeyDER,
		KeyId:     &m.keyID,
	}, nil
}

func TestNewKmsSignerFromClient_Success(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{pubKeyDER: der, keyID: awsTestKeyID}
	s, err := kmsSignerFromClient(t.Context(), mock, awsTestKeyID, nil)
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

func TestNewKmsSignerFromClient_Validation(t *testing.T) {
	specs := map[string]struct {
		client     KMSClient
		keyID      string
		errSubstr  string
	}{
		"empty key id": {
			client:    &mockKMSClient{},
			keyID:     "",
			errSubstr: "key ID is required",
		},
		"nil client": {
			client:    nil,
			keyID:     awsTestKeyID,
			errSubstr: "client is required",
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			_, err := kmsSignerFromClient(t.Context(), spec.client, spec.keyID, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), spec.errSubstr)
		})
	}
}

func TestNewKmsSignerFromClient_GetPublicKeyFails(t *testing.T) {
	mock := &mockKMSClient{
		getPubFn: func(_ context.Context, _ *kms.GetPublicKeyInput) (*kms.GetPublicKeyOutput, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	_, err := kmsSignerFromClient(t.Context(), mock, "test-key", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestSign_Success(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	expectedSig := []byte("test-signature-bytes")
	mock := &mockKMSClient{
		keyID:     awsTestKeyID,
		pubKeyDER: der,
		signFn: func(_ context.Context, params *kms.SignInput) (*kms.SignOutput, error) {
			keyID := awsTestKeyID
			assert.Equal(t, types.MessageTypeRaw, params.MessageType)
			assert.Equal(t, types.SigningAlgorithmSpecEd25519Sha512, params.SigningAlgorithm)
			return &kms.SignOutput{Signature: expectedSig, KeyId: &keyID}, nil
		},
	}

	s, err := kmsSignerFromClient(t.Context(), mock, awsTestKeyID, nil)
	require.NoError(t, err)

	sig, err := s.Sign(t.Context(), []byte("hello world"))
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
			signErr:      &smithy.GenericAPIError{Code: "ThrottlingException", Message: "rate limit"},
			errSubstr:    "AWS KMS sign failed",
			expectedCall: 4,
		},
		"max retries zero disables retries": {
			opts:         &Options{MaxRetries: 0},
			signErr:      &smithy.GenericAPIError{Code: "ThrottlingException", Message: "rate limit"},
			errSubstr:    "AWS KMS sign failed",
			expectedCall: 1,
		},
		"non retryable fails fast": {
			opts:         &Options{MaxRetries: 3},
			signErr:      fmt.Errorf("access denied"),
			errSubstr:    "non-retryable",
			expectedCall: 1,
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			_, der := generateTestEd25519DER(t)

			var calls int32
			signer := newTestSigner(t, &mockKMSClient{
				keyID:     awsTestKeyID,
				pubKeyDER: der,
				signFn: func(_ context.Context, _ *kms.SignInput) (*kms.SignOutput, error) {
					atomic.AddInt32(&calls, 1)
					return nil, spec.signErr
				},
			}, spec.opts)

			_, err := signer.Sign(t.Context(), []byte("hello world"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), spec.errSubstr)
			assert.Equal(t, spec.expectedCall, atomic.LoadInt32(&calls))
		})
	}
}

func TestSign_KeyIDMismatch_ReturnsError(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	unexpectedKeyID := "other-key"
	mock := &mockKMSClient{
		keyID:     awsTestKeyID,
		pubKeyDER: der,
		signFn: func(_ context.Context, _ *kms.SignInput) (*kms.SignOutput, error) {
			return &kms.SignOutput{
				Signature: []byte("test-signature-bytes"),
				KeyId:     &unexpectedKeyID,
			}, nil
		},
	}

	s := newTestSigner(t, mock, nil)

	_, err := s.Sign(t.Context(), []byte("hello world"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected key ID")
}

func TestGetPublic_Cached(t *testing.T) {
	pub, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{pubKeyDER: der, keyID: awsTestKeyID}
	s := newTestSigner(t, mock, nil)

	cryptoPub, err := s.GetPublic()
	require.NoError(t, err)

	raw, err := cryptoPub.Raw()
	require.NoError(t, err)
	assert.Equal(t, []byte(pub), raw)
}

func TestGetAddress_Deterministic(t *testing.T) {
	_, der := generateTestEd25519DER(t)

	mock := &mockKMSClient{pubKeyDER: der, keyID: awsTestKeyID}
	s := newTestSigner(t, mock, nil)

	addr1, err := s.GetAddress()
	require.NoError(t, err)

	addr2, err := s.GetAddress()
	require.NoError(t, err)

	assert.Equal(t, addr1, addr2, "address should be deterministic")
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

func newTestSigner(t *testing.T, mock *mockKMSClient, opts *Options) *KmsSigner {
	t.Helper()
	s, err := kmsSignerFromClient(t.Context(), mock, awsTestKeyID, opts)
	require.NoError(t, err)
	return s
}
