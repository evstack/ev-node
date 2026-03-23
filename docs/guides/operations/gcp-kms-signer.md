# Use GCP KMS Signer

Use this guide to run `ev-node` with a Google Cloud KMS-backed signer (`signer_type: kms`, `kms.provider: gcp`) instead
of a local key file.

## Prerequisites

- A Google Cloud KMS asymmetric key version with:
  - `purpose: ASYMMETRIC_SIGN`
  - `algorithm: EC_SIGN_ED25519`
- IAM permissions for runtime:
  - `cloudkms.cryptoKeyVersions.useToSign`
  - `cloudkms.cryptoKeyVersions.viewPublicKey`
- Google credentials available to the node process:
  - Application Default Credentials (recommended), or
  - service-account JSON file.

## 1. Create an ED25519 KMS key (example)

```bash
gcloud kms keyrings create ev-node --location=global

gcloud kms keys create signer \
  --location=global \
  --keyring=ev-node \
  --purpose=asymmetric-signing \
  --default-algorithm=ec-sign-ed25519
```

Get key version resource name:

```bash
gcloud kms keys versions list \
  --location=global \
  --keyring=ev-node \
  --key=signer
```

Use a full key version name like:
`projects/<project>/locations/global/keyRings/ev-node/cryptoKeys/signer/cryptoKeyVersions/1`

Create and export credentials for a service account:

```sh
# create
gcloud iam service-accounts create "kms-go-client" \
    --display-name "KMS Go Client Account"

# set permissions
gcloud projects add-iam-policy-binding <project> \
    --member "serviceAccount:kms-go-client@<project>.iam.gserviceaccount.com" \
    --role "roles/cloudkms.publicKeyViewer" \
    --role "roles/cloudkms.signerVerifier"

# export credentials file
gcloud iam service-accounts keys create /path/to/service-account.json \
    --iam-account "kms-go-client@<project>.iam.gserviceaccount.com"
```

## 2. Configure `evnode.yaml`

```yaml
signer:
  signer_type: "kms"
  kms:
    provider: "gcp"
    gcp:
      key_name: "projects/my-project/locations/global/keyRings/ev-node/cryptoKeys/signer/cryptoKeyVersions/1"
      credentials_file: "/path/to/service-account.json" # optional; ADC is used when omitted
      timeout: "1s"                                     # must be > 0
      max_retries: 3                                    # must be >= 0
```

## 3. Start as an aggregator

```bash
evnode start --evnode.node.aggregator
```

You should see a startup log line:

`initialized GCP KMS signer via factory`

## Troubleshooting

- `evnode.signer.kms.gcp.key_name is required when signer.signer_type is kms and signer.kms.provider is gcp`:
  Set `signer.kms.gcp.key_name`.
- `unsupported key type from KMS: expected ed25519`:
  Recreate key with `EC_SIGN_ED25519`.
- `KMS Sign failed ...`:
  Check IAM permissions, credentials, and network access to Cloud KMS.
