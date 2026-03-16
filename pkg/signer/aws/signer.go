// Package aws implements a signer.Signer backed by AWS KMS.
// It delegates signing to a remote KMS key and caches the public key locally.
package aws

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/evstack/ev-node/pkg/signer"
)

// KMSClient is the subset of the AWS KMS client API that KmsSigner needs.
// This allows mocking in tests.
type KMSClient interface {
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
}

// KmsSigner implements the signer.Signer interface using AWS KMS.
type KmsSigner struct {
	client    KMSClient
	keyID     string
	mu        sync.RWMutex
	publicKey crypto.PubKey
	address   []byte
}

var _ signer.Signer = (*KmsSigner)(nil)

// NewKmsSigner creates a new Signer backed by an AWS KMS Ed25519 key.
// It uses the standard AWS credential chain (env vars, ~/.aws/credentials, IAM roles, etc.).
func NewKmsSigner(ctx context.Context, region string, keyID string) (*KmsSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("aws kms key ID is required")
	}

	cfgOpts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := kms.NewFromConfig(cfg)
	return NewKmsSignerFromClient(ctx, client, keyID)
}

// NewKmsSignerFromClient creates a KmsSigner from an existing KMS client.
// Useful for testing with a mock client.
func NewKmsSignerFromClient(ctx context.Context, client KMSClient, keyID string) (*KmsSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("aws kms key ID is required")
	}

	s := &KmsSigner{
		client: client,
		keyID:  keyID,
	}

	// Fetch and cache the public key eagerly so we fail fast on misconfiguration.
	if err := s.fetchPublicKey(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch public key from KMS: %w", err)
	}

	return s, nil
}

// fetchPublicKey retrieves the public key from KMS and caches it.
func (s *KmsSigner) fetchPublicKey(ctx context.Context) error {
	out, err := s.client.GetPublicKey(ctx, &kms.GetPublicKeyInput{
		KeyId: aws.String(s.keyID),
	})
	if err != nil {
		return fmt.Errorf("KMS GetPublicKey failed: %w", err)
	}

	// AWS returns the public key as a DER-encoded X.509 SubjectPublicKeyInfo.
	pub, err := x509.ParsePKIXPublicKey(out.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse KMS public key: %w", err)
	}

	edPubKey, ok := pub.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("unsupported key type from KMS: expected ed25519, got %T", pub)
	}

	cryptoPubKey, err := crypto.UnmarshalEd25519PublicKey(edPubKey)
	if err != nil {
		return fmt.Errorf("failed to convert to libp2p pubkey: %w", err)
	}

	bz, err := cryptoPubKey.Raw()
	if err != nil {
		return fmt.Errorf("failed to get raw pubkey bytes: %w", err)
	}

	address := sha256.Sum256(bz)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.publicKey = cryptoPubKey
	s.address = address[:]

	return nil
}

// Sign signs a message using the remote KMS key.
func (s *KmsSigner) Sign(message []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := s.client.Sign(ctx, &kms.SignInput{
		KeyId:            aws.String(s.keyID),
		Message:          message,
		MessageType:      types.MessageTypeRaw,
		SigningAlgorithm: types.SigningAlgorithmSpecEd25519Sha512,
	})
	if err != nil {
		return nil, fmt.Errorf("KMS Sign failed: %w", err)
	}

	return out.Signature, nil
}

// GetPublic returns the cached public key.
func (s *KmsSigner) GetPublic() (crypto.PubKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.publicKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}

	return s.publicKey, nil
}

// GetAddress returns the cached address derived from the public key.
func (s *KmsSigner) GetAddress() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.address == nil {
		return nil, fmt.Errorf("address not loaded")
	}

	return s.address, nil
}
