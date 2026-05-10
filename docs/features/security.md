# Security

GitOpsCTL is designed with production security in mind, providing mechanisms to restrict controller access and manage secrets securely.

## Namespace-Scoped Security

By default, GitOpsCTL can apply manifests to any namespace defined in the kubeconfig. To enforce strict boundaries (e.g., in multi-tenant clusters), you can restrict a cluster's scope to specific namespaces.

### Configuration

Add the `allowedNamespaces` field to your cluster configuration in `configs/clusters.json`:

```json
{
  "name": "staging-cluster",
  "kubeconfigPath": "/etc/gitopsctl/kubeconfig",
  "allowedNamespaces": ["staging-apps", "monitoring"]
}
```

### Enforcement

When `allowedNamespaces` is defined:
- **Apply Blocked**: Any manifest targeting a namespace not in the list will be rejected before reaching the Kubernetes API.
- **Health Checks Restricted**: The controller will only assess the health of resources in allowed namespaces.
- **Resource Discovery**: Cluster-wide resources (like Namespaces or ClusterRoles) are still accessible if needed, but namespaced resources are strictly guarded.

## Secret Management (SOPS)

GitOpsCTL has native support for [Mozilla SOPS](https://github.com/getsops/sops). This allows you to store encrypted secrets directly in Git.

### How it Works

1. **Detection**: GitOpsCTL automatically detects SOPS-encrypted files (YAML/JSON) during the reconciliation loop.
2. **On-the-fly Decryption**: Files are decrypted in temporary memory before being applied to the cluster.
3. **Idempotency**: The controller only decrypts files when necessary, ensuring minimal overhead.

### Supported Providers

GitOpsCTL supports all standard SOPS providers:
- **Cloud KMS**: AWS KMS, GCP KMS, Azure Key Vault.
- **PGP**: For GPG-based encryption.
- **Age**: For modern, simple encryption.

For detailed setup instructions, see the [SOPS Documentation](../SOPS.md).

## Best Practices

1. **Principle of Least Privilege**: Use a service account with minimal RBAC permissions for the GitOpsCTL controller.
2. **Kubeconfig Isolation**: If running in Docker or on a server, use a dedicated kubeconfig file that only contains the necessary cluster contexts.
3. **Audit Monitoring**: Regularly review the [Audit Logs](observability.md) for unauthorized sync attempts or configuration changes.
