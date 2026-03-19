# Use AWS KMS Signer

Use this guide to run `ev-node` with an AWS KMS-backed signer (`signer_type: awskms`) instead of a local key file.

## Prerequisites

- An AWS KMS **asymmetric** key with:
  - `KeyUsage: SIGN_VERIFY`
  - `KeySpec: ECC_NIST_EDWARDS25519`
- IAM permissions for initial key creation/management (example policy):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowKeyCreation",
      "Effect": "Allow",
      "Action": [
        "kms:CreateKey",
        "kms:TagResource",
        "kms:EnableKey",
        "kms:PutKeyPolicy",
        "kms:GetPublicKey",
        "kms:Sign",
        "kms:ListKeys",
        "kms:ListAliases"
      ],
      "Resource": "*"
    }
  ]
}
```

- Runtime IAM permissions for `ev-node` (minimum):
  - `kms:GetPublicKey`
  - `kms:Sign`
- AWS credentials available to the node process (IAM role, env vars, or shared profile).

## 1. Create an ED25519 KMS key (example)

```bash
aws kms create-key \
  --description "ev-node signer" \
  --key-usage SIGN_VERIFY \
  --key-spec ECC_NIST_EDWARDS25519
```

Copy the returned key ARN (or key ID). You can also create an alias and use that.

## 2. Configure `evnode.yaml`

```yaml
signer:
  signer_type: "awskms"
  kms_key_id: "arn:aws:kms:us-east-1:123456789012:key/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  kms_region: "us-east-1"    # optional but recommended
  kms_profile: "prod"        # optional; omit when using IAM role/env creds
  kms_timeout: "10s"         # must be > 0
  kms_max_retries: 3         # must be >= 0
```

## 3. Start as an aggregator

```bash
evnode start --evnode.node.aggregator
```

You should see a startup log line:

`initialized AWS KMS signer via factory`

## Troubleshooting

- `evnode.signer.kms_key_id is required when signer_type is awskms`:
  Set `signer.kms_key_id`.
- `unsupported key type from KMS: expected ed25519`:
  Recreate the key as `ECC_NIST_EDWARDS25519`.
- `KMS Sign failed ...`:
  Check IAM permissions, key policy, region/profile, and network access to AWS KMS.
