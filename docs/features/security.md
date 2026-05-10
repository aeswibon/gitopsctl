# Security

GitOpsCTL can manage powerful Kubernetes credentials, so production setups should treat it like any other deployment controller.

## Kubeconfig Security

Use a dedicated kubeconfig for GitOpsCTL instead of a personal admin kubeconfig.

Recommended practices:

- Use a dedicated Kubernetes service account.
- Grant only the verbs and resources GitOpsCTL needs.
- Scope permissions by namespace when possible.
- Store kubeconfig files outside the repository.
- Mount kubeconfigs read-only in containers.
- Rotate credentials regularly.

## Namespace Restrictions

GitOpsCTL supports an application-layer namespace guard through the cluster `allowedNamespaces` field.

```json
[
  {
    "name": "staging",
    "kubeconfigPath": "/etc/gitopsctl/kubeconfig-staging",
    "allowedNamespaces": ["staging", "monitoring"]
  }
]
```

You can also set it with the CLI:

```bash
gitopsctl register-cluster \
  --name staging \
  --kubeconfig /etc/gitopsctl/kubeconfig-staging \
  --allowed-namespaces staging,monitoring
```

Behavior:

- Empty `allowedNamespaces` means no GitOpsCTL namespace restriction.
- Namespaced resources without `metadata.namespace` default to `default`.
- Namespaced resources outside `allowedNamespaces` are rejected before apply.
- Cluster-scoped resources are not namespace-scoped, so protect them with Kubernetes RBAC.

This guard complements Kubernetes RBAC. It does not replace RBAC.

## RBAC Example

A minimal namespace-scoped role depends on the resources your apps manage. Start narrow and expand intentionally.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gitopsctl
  namespace: demo
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: gitopsctl-applier
  namespace: demo
rules:
- apiGroups: ["", "apps", "batch", "networking.k8s.io"]
  resources: ["configmaps", "secrets", "services", "deployments", "statefulsets", "daemonsets", "jobs", "cronjobs", "ingresses"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: gitopsctl-applier
  namespace: demo
subjects:
- kind: ServiceAccount
  name: gitopsctl
  namespace: demo
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: gitopsctl-applier
```

If your manifests create namespaces or cluster-scoped resources, you need explicit cluster-level RBAC. Keep that separate from normal app deployment credentials when possible.

## Secret Management With SOPS

GitOpsCTL decrypts SOPS-encrypted `.yaml`, `.yml`, and `.json` files during reconciliation when the runtime environment has access to the required key material.

Supported SOPS providers include:

- Age
- PGP
- AWS KMS
- GCP KMS
- Azure Key Vault
- HashiCorp Vault, when configured through SOPS

See [SOPS Secret Management](../SOPS.md) for setup details.

## Webhook Security

For controller event webhooks:

```bash
gitopsctl start \
  --events-webhook https://example.com/gitopsctl/events \
  --events-webhook-bearer "$TOKEN" \
  --events-webhook-secret "$SIGNING_SECRET"
```

Use HTTPS endpoints, short-lived tokens when possible, and verify HMAC signatures on the receiver.

## Configuration Hygiene

Do not commit:

- Kubeconfig files.
- Decrypted SOPS secrets.
- Webhook signing secrets.
- Cloud provider credentials.
- Private Age or PGP keys.

It is safe to commit encrypted SOPS manifests and non-sensitive sample configs.
