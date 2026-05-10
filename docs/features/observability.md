# Observability

GitOpsCTL provides comprehensive observability through Prometheus metrics, persistent audit logs, and customizable webhooks.

## Prometheus Metrics

GitOpsCTL exposes a Prometheus-compatible `/metrics` endpoint on the API server (default port `:8080`).

### Key Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `gitopsctl_app_sync_total` | Counter | `app`, `cluster`, `status` | Total sync attempts. |
| `gitopsctl_cluster_status` | Gauge | `cluster` | 1 for Healthy, 0 for Error. |
| `gitopsctl_app_health_status` | Gauge | `app`, `cluster` | 1=Healthy, 0.5=Progressing, 0=Degraded. |
| `gitopsctl_k8s_apply_total` | Counter | `app`, `cluster`, `kind`, `status` | Individual resource apply operations. |
| `gitopsctl_git_pull_total` | Counter | `app`, `status` | Git clone/pull operations. |
| `gitopsctl_app_sync_duration_seconds` | Histogram | `app`, `cluster` | Time taken for successful syncs. |

## Audit Logs (JSONL)

You can enable persistent audit logs by providing the `--events-file` flag when starting the controller. Every significant event is appended to this file as a JSON line.

```bash
gitopsctl start --events-file /var/log/gitopsctl/audit.jsonl
```

Example log entry:
```json
{"type":"io.gitopsctl.app.sync.succeeded","time":"2023-10-27T10:00:00Z","data":{"app":"my-app","cluster":"prod","hash":"a1b2c3d"}}
```

## Webhooks

GitOpsCTL can POST integration events to an external URL.

```bash
gitopsctl start \
  --events-webhook https://api.my-dashboard.com/webhooks \
  --events-webhook-secret my-hmac-secret
```

### Security
If `--events-webhook-secret` is provided, GitOpsCTL will include a `X-GitOpsctl-Signature` header in the request. This signature is an HMAC-SHA256 hash of the JSON payload.

### Webhook Reliability
- **Retries**: Configurable via `--events-webhook-retries` (default 2).
- **Backoff**: Configurable via `--events-webhook-backoff` (default 750ms).
- **Timeout**: Configurable via `--events-webhook-timeout` (default 12s).
