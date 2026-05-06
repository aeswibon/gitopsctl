# Integrations — events and dashboards

GitOpsCTL is **CLI-first**. For custom dashboards and automation, consume **integration events** instead of scraping unstructured logs.

Delivery guarantees and compatibility policy are defined in [ADR 0001](./adr/0001-events-delivery-and-compatibility.md).

## Enabling events

When you run the controller:

```bash
gitopsctl start \
  --events-file configs/events.jsonl \
  --events-webhook https://your-receiver.example/hooks/gitopsctl \
  --events-webhook-bearer "$TOKEN" \
  --events-webhook-secret "$HMAC_SECRET" \
  --events-webhook-retries 3 \
  --events-webhook-backoff 1s
```

- **`--events-file`**: append-only JSON Lines (one JSON object per line). Safe to `tail -f` or use `gitopsctl tail-events`.
- **`--events-webhook`**: each event is POSTed as `application/json`.
- **`--events-webhook-bearer`**: optional `Authorization: Bearer …` header for the webhook.
- **`--events-webhook-secret`**: optional HMAC signing key for tamper verification.
- **`--events-webhook-retries`**: transient retry attempts (network errors, HTTP `5xx`, `429`).
- **`--events-webhook-backoff`**: exponential backoff base duration between retries.
- **`--events-webhook-timeout`**: per-request HTTP timeout.

You can enable **one or both** sinks.

## Event envelope (version 1.0)

Each record matches this shape:

| Field | Type | Description |
|-------|------|-------------|
| `specversion` | string | Always `1.0` for this schema. |
| `id` | string | Unique id (UUID) per delivery. |
| `type` | string | Stable event type (see below). |
| `source` | string | Usually `gitopsctl-controller` (runtime) or `gitopsctl-cli` (mutating CLI commands with event sinks configured). |
| `time` | string | UTC RFC3339 timestamp. |
| `data` | object | Event-specific payload. |

## Event types

| `type` | When | Typical `data` fields |
|--------|------|-------------------------|
| `io.gitopsctl.controller.started` | After controller dispatches initial reconcilers | `applications`, `clusters` (counts) |
| `io.gitopsctl.controller.stopping` | Before shutdown begins | (often empty) |
| `io.gitopsctl.app.registered` | App registered or updated by API/CLI | `app`, `repoURL`, `branch`, `path`, `cluster`, `interval`, `updated` |
| `io.gitopsctl.app.unregistered` | App removed by API/CLI | `app` (+ optional metadata in CLI path) |
| `io.gitopsctl.app.sync_requested` | Manual sync requested (API or `sync-app`) | `app` |
| `io.gitopsctl.app.sync_started` | Start of a sync attempt | `app`, `cluster`, `trigger` (`initial` \| `poll` \| `manual`), `lastSyncedHash` |
| `io.gitopsctl.app.git_pull_failed` | Git clone/pull error | `app`, `cluster`, `trigger`, `error` |
| `io.gitopsctl.app.manifest_path_missing` | Manifest path missing in repo | `app`, `cluster`, `trigger`, `path` |
| `io.gitopsctl.app.apply_failed` | kubectl apply errors | `app`, `cluster`, `trigger`, `error` |
| `io.gitopsctl.app.sync_succeeded` | Manifests applied and commit recorded | `app`, `cluster`, `trigger`, `commit`, `previousCommit` |
| `io.gitopsctl.app.sync_no_changes` | Repo already at synced commit | `app`, `cluster`, `trigger`, `commit` — **not** emitted on periodic polls when nothing changed (only `manual` / `initial`) |

Cluster connectivity:

| `type` | When | Typical `data` fields |
|--------|------|-------------------------|
| `io.gitopsctl.cluster.registered` | Cluster registered or updated by API/CLI | `cluster`, `kubeconfig` |
| `io.gitopsctl.cluster.unregistered` | Cluster removed by API/CLI | `cluster` |
| `io.gitopsctl.cluster.health_check_requested` | Manual health check requested | `cluster` |
| `io.gitopsctl.cluster.health_check_completed` | After each health check run | `cluster`, `status`, `message` |

## CLI parity with the API

These commands call the **running controller** HTTP API (same as curl):

- `gitopsctl sync-app -n <app>` → `POST /api/v1/applications/:name/sync`
- `gitopsctl check-cluster -n <name>` → `POST /api/v1/clusters/:name/check`

Use `--api-url` on the root command if the API is not at `http://127.0.0.1:8080`.

## Follow events locally

```bash
gitopsctl tail-events --file configs/events.jsonl
```

## Live stream (SSE)

For browser-friendly or backend subscribers that prefer a long-lived stream:

```bash
curl -N http://127.0.0.1:8080/api/v1/events
```

SSE frames include:

- `id`: event envelope id
- `event`: event type
- `data`: full JSON envelope

This stream is best-effort in-memory fan-out. Slow clients may miss events and should tolerate gaps.

## CLI scenario test scripts

- `scripts/cli-smoke.sh` — broad command coverage + synthetic/runtime checks; safe for CI.
- `scripts/cli-real-scenario.sh` — strict real-cluster scenario (fails hard on errors), intended for local/release validation.

## Semantics

- **Ordering**: no strict global ordering guarantee across apps; file sink preserves append order per process.
- **Delivery**: webhook posts retry transient failures using exponential backoff, then fail best-effort if attempts are exhausted. Build idempotent receivers keyed by `id`.
- **Security**: treat webhook URLs, bearer tokens, and signing secrets as secrets; prefer HTTPS.

## Webhook signing

When `--events-webhook-secret` is set, GitOpsCTL signs each request with:

- `X-GitOpsctl-Timestamp`: RFC3339Nano UTC timestamp
- `X-GitOpsctl-Event-ID`: envelope id
- `X-GitOpsctl-Signature`: `sha256=<hex hmac>`

Signature payload:

```text
<timestamp>.<raw JSON body>
```

Use the same HMAC secret on the receiver to verify authenticity and integrity.

### Go verification example

```go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func verifyWebhook(secret string, r *http.Request) ([]byte, error) {
	sigHeader := r.Header.Get("X-GitOpsctl-Signature") // sha256=<hex>
	tsHeader := r.Header.Get("X-GitOpsctl-Timestamp")  // RFC3339Nano
	if sigHeader == "" || tsHeader == "" {
		return nil, fmt.Errorf("missing signature headers")
	}

	parts := strings.SplitN(sigHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return nil, fmt.Errorf("invalid signature format")
	}
	gotSig, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad signature hex: %w", err)
	}

	// Optional replay protection window (5m).
	ts, err := time.Parse(time.RFC3339Nano, tsHeader)
	if err != nil {
		return nil, fmt.Errorf("bad timestamp: %w", err)
	}
	if d := time.Since(ts); d > 5*time.Minute || d < -30*time.Second {
		return nil, fmt.Errorf("timestamp outside allowed window")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(tsHeader))
	mac.Write([]byte("."))
	mac.Write(body)
	wantSig := mac.Sum(nil)

	if subtle.ConstantTimeCompare(gotSig, wantSig) != 1 {
		return nil, fmt.Errorf("signature mismatch")
	}
	return body, nil
}
```

Use `X-GitOpsctl-Event-ID` for idempotency (dedupe repeated deliveries).
