# SOPS Secret Management in GitOpsCTL

GitOpsCTL provides native support for [SOPS (Secrets Operations)](https://github.com/getsops/sops), allowing you to store encrypted secrets in your Git repository.

## How it Works

The GitOpsCTL controller automatically detects SOPS-encrypted files in your application's manifest directory. During each synchronization:

1. The repository is cloned/pulled to a temporary directory.
2. The controller walks through the directory and identifies encrypted files (`.yaml`, `.yml`, `.json`).
3. Files containing SOPS metadata are decrypted in-place using the SOPS library.
4. The decrypted manifests are then applied to the Kubernetes cluster.
5. The temporary directory is cleaned up immediately after synchronization.

## Supported Providers

Since GitOpsCTL uses the SOPS library directly, it supports all providers that SOPS supports, provided the environment is configured correctly on the host running the controller:

- **PGP**: Ensure GPG is installed and the private key is in the keyring.
- **AWS KMS**: Ensure the controller has AWS credentials with `kms:Decrypt` permissions.
- **GCP KMS**: Ensure the controller has GCP credentials with `cloudkms.cryptoKeyVersions.useToDecrypt` permissions.
- **Azure Key Vault**: Ensure the controller is authenticated with Azure.
- **Age**: Ensure the `SOPS_AGE_KEY_FILE` or `SOPS_AGE_KEY` environment variables are set.

## Configuration

No special configuration is needed in GitOpsCTL. As long as your files are encrypted with SOPS and the environment where the controller runs has access to the decryption keys, it will work automatically.

### Example: Encrypting a Secret

```bash
# Encrypt a secret using a PGP key
sops --encrypt --pgp <YOUR_PGP_FINGERPRINT> secret.yaml > secret.enc.yaml

# Encrypt using AWS KMS
sops --encrypt --kms <YOUR_KMS_ARN> secret.yaml > secret.enc.yaml
```

## Security Best Practices

1. **Least Privilege**: Ensure the controller's identity (e.g., IAM Role, ServiceAccount) only has the minimum permissions required to decrypt the specific keys used for your GitOps repo.
2. **Key Rotation**: Regularly rotate your encryption keys. SOPS makes it easy to re-encrypt files with new keys.
3. **No Plaintext in Git**: Never commit decrypted secrets to your Git repository. Always use SOPS to encrypt them before pushing.
