# AWS KMS Signer

This package implements `signer.Signer` using AWS KMS.

It uses KMS for `Sign` operations and caches the public key/address in memory after initialization.

## Requirements

- AWS credentials must be available via the standard AWS SDK credential chain.
- The configured KMS key must be an asymmetric **Ed25519** key.

## Configuration

Set `evnode.signer.signer_type` to `kms`, set `evnode.signer.kms.provider` to `aws`,
and provide at least `evnode.signer.kms.aws.key_id`.

Example:

```yaml
signer:
  signer_type: kms
  kms:
    provider: aws
    aws:
      key_id: arn:aws:kms:eu-central-1:123456789012:key/00000000-0000-0000-0000-000000000000
      region: eu-central-1
      profile: default
      timeout: 1s
      max_retries: 3
```

## Notes

- `kms.aws.timeout` is the timeout per KMS Sign request.
- `kms.aws.max_retries` controls retries for transient KMS/API/network failures.
