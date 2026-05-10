# Observability

GitOpsCTL exposes runtime state through CLI status commands, the dashboard, Prometheus metrics, Server-Sent Events, JSONL audit logs, and HTTP webhooks.

## Status Commands

```bash
gitopsctl list-apps
gitopsctl status-apps
gitopsctl list-clusters
gitopsctl status-clusters
```

API-backed commands need a running controller:

```bash
gitopsctl --api-url http://127.0.0.1:8080 sync-app --name nginx-demo
gitopsctl --api-url http://127.0.0.1:8080 check-cluster --name local-dev
```

## Dashboard

```bash
gitopsctl dashboard --api-url http://127.0.0.1:8080
```

The dashboard reads app and cluster data from the REST API and refreshes on SSE events.

## Prometheus Metrics

The API server exposes Prometheus metrics on the same listen address as the REST API.

```bash
curl http://127.0.0.1:8080/metrics
```

Key metrics:

| Metric | Type | Labels | Meaning |
|--------|------|--------|---------|
| `gitopsctl_app_sync_total` | Counter | `app`, `cluster`, `status` | Application sync attempts by result. |
| `gitopsctl_cluster_status` | Gauge | `cluster` | `1` for reachable, `0` for unreachable/error. |
| `gitopsctl_app_sync_duration_seconds` | Histogram | `app`, `cluster` | Successful sync duration. |
| `gitopsctl_app_health_status` | Gauge | `app`, `cluster` | `1` healthy, `0.5` progressing, `0` degraded/error. |
| `gitopsctl_k8s_apply_total` | Counter | `app`, `cluster`, `kind`, `status` | Kubernetes resource apply operations. |
| `gitopsctl_git_pull_total` | Counter | `app`, `status` | Git clone/pull operations. |

## JSONL Event Log

Enable file events:

```bash
gitopsctl start --events-file configs/events.jsonl
```

Follow events:

```bash
gitopsctl tail-events --file configs/events.jsonl --from-start
```

Each line is a JSON event envelope. This is useful for local audit trails, ingestion into log pipelines, and debugging reconciliation.

## Webhooks

Enable event webhooks:

```bash
gitopsctl start \
  --events-webhook https://example.com/gitopsctl/events \
  --events-webhook-bearer "$TOKEN" \
  --events-webhook-secret "$SIGNING_SECRET" \
  --events-webhook-retries 3 \
  --events-webhook-backoff 1s \
  --events-webhook-timeout 10s
```

When a signing secret is set, webhook requests include an HMAC SHA-256 signature header. Receivers should verify the signature before trusting the payload.

## Server-Sent Events

The dashboard uses the SSE stream exposed by the API server. You can inspect it manually:

```bash
curl -N http://127.0.0.1:8080/api/v1/events
```

SSE is intended for live local or internal consumers. Use JSONL or webhooks for durable external processing.

## Operational Checks

For a healthy controller:

- `/health` responds successfully.
- `/metrics` returns Prometheus text.
- `status-clusters` shows recent cluster checks.
- `status-apps` shows recent sync status and commit hashes.
- `tail-events` shows controller and sync events when event logging is enabled.
