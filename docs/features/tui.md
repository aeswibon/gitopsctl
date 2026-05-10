# Terminal Dashboard

The dashboard is an interactive terminal UI built with Bubble Tea and Lipgloss. It connects to the GitOpsCTL API server and provides a live view of applications and clusters.

## Start the Dashboard

Start the controller first:

```bash
gitopsctl start --api-address :8080
```

Open the dashboard in another terminal:

```bash
gitopsctl dashboard --api-url http://127.0.0.1:8080
```

## Views

### Applications

Shows registered applications, current sync status, cluster assignment, interval, commit hash, failure count, and message.

Common statuses:

- `Synced`: Last discovered/approved commit was applied.
- `Healthy`: Applied resources are currently healthy.
- `Progressing`: Applied resources are still converging.
- `Degraded`: At least one applied resource is unhealthy.
- `OutOfSync`: A newer commit exists but has not been applied, usually because manual approval is required.
- `Pending`: App is registered but has not completed reconciliation.
- `Error`: Git, render, Kubernetes, or policy failure occurred.
- `Stopped`: Reconciliation stopped because the controller shut down or the app was stopped.

### Clusters

Shows registered clusters, kubeconfig path, connectivity status, last check time, and status message.

Common statuses:

- `Active`: Cluster connectivity check succeeded.
- `Unreachable`: API server connection or discovery failed.
- `Pending`: Cluster is registered and awaiting validation.
- `Error`: Client creation or configuration failed.

## Keyboard Controls

| Key | Action |
|-----|--------|
| `tab`, `shift+tab` | Switch between applications and clusters. |
| `up`, `k` | Move selection up. |
| `down`, `j` | Move selection down. |
| `r` | Refresh app and cluster data. |
| `s` | Request sync for selected application. |
| `c` | Request health check for selected cluster. |
| `u` | Unregister selected application or cluster. |
| `y` | Confirm pending action. |
| `n`, `esc` | Cancel pending action. |
| `q`, `ctrl+c` | Quit. |

## Live Updates

The dashboard starts by fetching application and cluster lists through the REST API. It then listens for Server-Sent Events and refreshes when the controller emits changes.

If the dashboard cannot connect:

1. Confirm the controller is running.
2. Confirm `--api-url` matches the controller `--api-address`.
3. Try `curl http://127.0.0.1:8080/health`.
4. Check firewall or container port mappings.

## When to Use CLI Instead

The dashboard is ideal for live operations. Use CLI commands for scripts, CI, and repeatable workflows:

```bash
gitopsctl status-apps
gitopsctl sync-app --name nginx-demo
gitopsctl approve-app --name nginx-demo --commit <commit-hash>
gitopsctl check-cluster --name local-dev
```
