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

// Options configures optional KmsSigner behaviour.
type Options struct {
	// Timeout for individual KMS Sign API calls. Default: 10s.
	Timeout time.Duration
	// MaxRetries for transient KMS failures during Sign. Default: 3.
	MaxRetries int
	// CacheTTL controls how long the public key is cached before being
	// re-fetched from KMS. 0 (default) means cache forever.
	CacheTTL time.Duration
}

func (o *Options) timeout() time.Duration {
	if o != nil && o.Timeout > 0 {
		return o.Timeout
	}
	return 10 * time.Second
}

func (o *Options) maxRetries() int {
	if o != nil && o.MaxRetries > 0 {
		return o.MaxRetries
	}
	return 3
}

func (o *Options) cacheTTL() time.Duration {
	if o != nil {
		return o.CacheTTL
	}
	return 0
}

// KmsSigner implements the signer.Signer interface using AWS KMS.
type KmsSigner struct {
	client  KMSClient
	keyID   string
	opts    Options
	mu      sync.RWMutex
	pubKey  crypto.PubKey
	address []byte
	cacheAt time.Time
}

var _ signer.Signer = (*KmsSigner)(nil)

// NewKmsSigner creates a new Signer backed by an AWS KMS Ed25519 key.
// It uses the standard AWS credential chain (env vars, ~/.aws/credentials, IAM roles, etc.).
func NewKmsSigner(ctx context.Context, region string, profile string, keyID string, opts *Options) (*KmsSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("aws kms key ID is required")
	}

	cfgOpts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithRegion(region))
	}
	if profile != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithSharedConfigProfile(profile))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := kms.NewFromConfig(cfg)
	return NewKmsSignerFromClient(ctx, client, keyID, opts)
}

// NewKmsSignerFromClient creates a KmsSigner from an existing KMS client.
// Useful for testing with a mock client.
func NewKmsSignerFromClient(ctx context.Context, client KMSClient, keyID string, opts *Options) (*KmsSigner, error) {
	if keyID == "" {
		return nil, fmt.Errorf("aws kms key ID is required")
	}

	o := Options{}
	if opts != nil {
		o = *opts
	}

	s := &KmsSigner{
		client: client,
		keyID:  keyID,
		opts:   o,
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

	s.pubKey = cryptoPubKey
	s.address = address[:]
	s.cacheAt = time.Now()

	return nil
}

// Sign signs a message using the remote KMS key with configurable timeout
// and retry with exponential backoff.
func (s *KmsSigner) Sign(ctx context.Context, message []byte) ([]byte, error) {
	var lastErr error
	maxRetries := s.opts.maxRetries()
	timeout := s.opts.timeout()

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms, ...
			backoff := time.Duration(100<<uint(attempt-1)) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("KMS Sign canceled: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		callCtx, cancel := context.WithTimeout(ctx, timeout)
		out, err := s.client.Sign(callCtx, &kms.SignInput{
			KeyId:            aws.String(s.keyID),
			Message:          message,
			MessageType:      types.MessageTypeRaw,
			SigningAlgorithm: types.SigningAlgorithmSpecEd25519Sha512,
		})
		cancel()

		if err != nil {
			lastErr = err
			continue
		}

		return out.Signature, nil
	}

	return nil, fmt.Errorf("KMS Sign failed after %d attempts: %w", maxRetries, lastErr)
}

// GetPublic returns the cached public key, optionally refreshing if cache TTL
// has expired.
func (s *KmsSigner) GetPublic() (crypto.PubKey, error) {
	ttl := s.opts.cacheTTL()

	s.mu.RLock()
	pubKey := s.pubKey
	expired := ttl > 0 && time.Since(s.cacheAt) > ttl
	s.mu.RUnlock()

	if expired {
		if err := s.fetchPublicKey(context.Background()); err != nil {
			// If refresh fails, return the stale cached key
			if pubKey != nil {
				return pubKey, nil
			}
			return nil, fmt.Errorf("failed to refresh public key: %w", err)
		}
		s.mu.RLock()
		pubKey = s.pubKey
		s.mu.RUnlock()
	}

	if pubKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}

	return pubKey, nil
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
