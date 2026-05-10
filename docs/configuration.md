# Configuration

GitOpsCTL uses JSON files in the `configs/` directory to manage applications and clusters.

## Application Configuration (`apps.json`)

Each application entry supports the following fields:

| Field | Description | Example |
|-------|-------------|---------|
| `name` | Unique name for the application. | `frontend-prod` |
| `repoUrl` | URL of the Git repository. | `https://github.com/org/repo.git` |
| `branch` | Git branch to watch. | `main` |
| `path` | Path within the repo to the manifests. | `deploy/prod` |
| `clusterName` | Name of the cluster to deploy to. | `prod-cluster` |
| `syncPolicy` | `auto` (apply immediately) or `manual` (require approval). | `auto` |
| `pollingInterval` | How often to check for Git changes. | `1m0s` |

### Example `apps.json`

```json
[
  {
    "name": "sample-app",
    "repoUrl": "https://github.com/aeswibon/gitops-examples",
    "branch": "main",
    "path": "manifests/sample-app",
    "clusterName": "local-cluster",
    "syncPolicy": "auto",
    "pollingInterval": "1m0s"
  }
]
```

## Cluster Configuration (`clusters.json`)

Each cluster entry supports the following fields:

| Field | Description | Example |
|-------|-------------|---------|
| `name` | Unique name for the cluster. | `local-cluster` |
| `kubeconfigPath` | Absolute path to the kubeconfig file. | `/home/user/.kube/config` |
| `allowedNamespaces` | (Optional) List of namespaces this cluster can touch. | `["default", "staging"]` |

### Example `clusters.json`

```json
[
  {
    "name": "local-cluster",
    "kubeconfigPath": "/Users/user/.kube/config",
    "allowedNamespaces": ["default", "app-namespace"]
  }
]
```

## Global Flags

When running `gitopsctl start`, you can use several global flags:

- `--api-address`: Address for the REST API (default `:8080`).
- `--events-file`: Path to a JSONL file for audit logs.
- `--events-webhook`: URL to post integration events to.
- `--events-webhook-bearer`: Bearer token for webhook authentication.
- `--events-webhook-secret`: HMAC secret for signing webhook payloads.

Example:
```bash
gitopsctl start --events-file /var/log/gitopsctl/audit.jsonl --api-address 0.0.0.0:8080
```
