# AWS KMS Signer

This package implements `signer.Signer` using AWS KMS.

It uses KMS for `Sign` operations and caches the public key/address in memory after initialization.

## Requirements

- AWS credentials must be available via the standard AWS SDK credential chain.
- The configured KMS key must be an asymmetric **Ed25519** key.

## Configuration

Set `evnode.signer.signer_type` to `awskms` and provide at least `kms_key_id`.

Example:

```yaml
signer:
  signer_type: awskms
  kms_key_id: arn:aws:kms:eu-central-1:123456789012:key/00000000-0000-0000-0000-000000000000
  kms_region: eu-central-1
  kms_profile: default
  kms_timeout: 1s
  kms_max_retries: 3
```

## Notes

- `kms_timeout` is the timeout per KMS Sign request.
- `kms_max_retries` controls retries for transient KMS/API/network failures.
