# GCP KMS Signer

This package implements `signer.Signer` using Google Cloud KMS.

It uses KMS for `AsymmetricSign` operations and caches the public key/address in memory after initialization.

## Requirements

- Google Cloud credentials must be available via Application Default Credentials (ADC), or
  `kms.gcp.credentials_file` must be set.
- The configured KMS key version must be an asymmetric **Ed25519** key version.

## Configuration

Set `evnode.signer.signer_type` to `kms`, set `evnode.signer.kms.provider` to `gcp`,
and provide at least `evnode.signer.kms.gcp.key_name`.

Example:

```yaml
signer:
  signer_type: kms
  kms:
    provider: gcp
    gcp:
      key_name: projects/my-project/locations/global/keyRings/ev-node/cryptoKeys/signer/cryptoKeyVersions/1
      credentials_file: /path/to/service-account.json
      timeout: 10s
      max_retries: 3
```

## Notes

- `kms.gcp.timeout` is the timeout per KMS Sign request.
- `kms.gcp.max_retries` controls retries for transient KMS/API/network failures.
