# Phase 2 scope — CLI-first product, integrations for dashboards

## Product stance

**GitOpsCTL is a CLI-first tool.** The binary is the canonical way to operate the controller, manage apps and clusters, and inspect state. Anything we ship should assume operators live in the terminal, scripts, and CI— not inside a vendor-specific web UI.

The optional HTTP server started with `gitopsctl start` is a **machine interface** for the running controller (automation, remote triggers, health probes). It is **not** the primary product surface and does not replace CLI completeness.

**We do not ship a first-party dashboard.** Teams who want charts, multi-user consoles, or custom workflows should **listen to GitOpsCTL events** and **call the CLI or APIs** from their own services. Phase 2 is where we make that intentional and complete.

## Principles

1. **CLI parity**: Every meaningful operation available while the controller runs should also be available from the CLI (including triggers today exposed only via HTTP). Output must be script-friendly (`--output json` / stable schemas where listing matters).

2. **Observable actions**: Important lifecycle and reconciliation events are emitted in a **stable, documented format** so external processes can subscribe—not buried only in human-oriented log lines.

3. **Thin integration layer**: Prefer webhooks, structured streams, or exec hooks over growing an embedded UI.

## Already helps dashboard builders (baseline)

- Structured zap logs when the controller runs (parseable with a log pipeline).
- REST routes under `/api/v1` for apps/clusters/sync/check (usable by a backend that powers a custom UI).
- CLI list/status commands with formatting flags.

Phase 2 closes gaps so **dashboard backends do not depend on scraping unstructured logs**.

## Phase 2 deliverables (recommended order)

### 1. CLI completeness and stable machine output

- Audit HTTP handlers vs CLI: add missing commands or flags so **no capability is API-only**.
- Standardize **JSON schemas** for list/status/get outputs; document them in-repo.
- Consider a single **`gitopsctl events tail`** (or **`gitopsctl watch`**) that streams JSON lines to stdout for local tooling—optional but high leverage for “glue” dashboards.

### 2. Event contract (for “listen and build your dashboard”)

Define a **small versioned event envelope**, for example:

- `specversion`, `type`, `source`, `time`, `data` (OpenTelemetry-style or CloudEvents-like—pick one and stick to it).

Illustrative **event types** (exact names TBD in an RFC):

- Controller lifecycle: started, stopping.
- App: registered, unregistered, sync_started, sync_succeeded, sync_failed, git_pull_failed, apply_failed.
- Cluster: registered, unregistered, health_check_completed.

### 3. Pluggable sinks (how listeners attach)

Implement **one or two** first-class sinks (avoid boiling the ocean):

| Sink | Use case |
|------|----------|
| **HTTP webhook** | User URLs receive POST with JSON events; simplest for SaaS dashboards or serverless handlers. |
| **Append-only JSONL file** | Cheap durability and tail -f for local dev or agents. |
| **Unix socket / TCP stream** (optional) | Low-latency consumers on the same host. |

Multiple sinks can be enabled from config (e.g. `configs/events.yaml` or flags on `start`).

### 4. Documentation for dashboard authors

- **Integration guide**: event types, payloads, retry semantics for webhooks, ordering guarantees (best-effort vs at-least-once—document honestly).
- Clarify security: webhook URLs often need TLS and shared secrets; document signing or static bearer tokens for callbacks.

### 5. REST API role in Phase 2

- Keep `/api/v1` as **optional** automation surface.
- Align HTTP payloads with CLI JSON where possible so one schema drives both.
- Optional later: **SSE or WebSocket** on `/api/v1/events` if we want browser-adjacent consumers without webhooks—explicitly secondary to CLI + webhooks.

## Explicitly out of scope for Phase 2

- Official GitOpsCTL web UI or hosted SaaS.
- Full Helm/OCI plugin ecosystem (can move to Phase 3 unless prioritized).
- Replacing polling with Git webhooks as the only sync path (can be Phase 2 stretch or Phase 3).

## Relationship to Phase 3

Phase 3 can focus on **sync strategies**, **Helm/OCI**, and deeper **engine plugins**—still without requiring a built-in dashboard, unless the community later decides otherwise via governance.

---

## Implementation status (ready to close)

Shipping in-tree:

- Integration **event envelope** (`specversion` 1.0) and controller emits for lifecycle, sync outcomes, and cluster health.
- **JSONL file** sink and optional **HTTP webhook** sink (`gitopsctl start` flags).
- **`gitopsctl tail-events`** for local JSONL following.
- **CLI ↔ API parity** for manual sync and cluster check: `sync-app`, `check-cluster` plus global **`--api-url`**.
- Register/unregister/requested events are emitted from both API handler flows and mutating CLI flows.
- Webhook hardening: retries + backoff + optional HMAC signing headers.

## Phase 2 completion checklist

- [x] CLI parity for API-only operational actions (`sync-app`, `check-cluster`)
- [x] Stable event envelope + documented event type catalog
- [x] At least two sinks (JSONL + webhook)
- [x] Dashboard integration docs, including webhook signature verification example
- [x] Webhook hardening (retry/backoff + optional signing)
- [x] Basic webhook behavior tests (retry path, non-retryable path, signing headers)
- [x] ADR for delivery guarantees and compatibility policy (versioning/deprecation) — [ADR 0001](./adr/0001-events-delivery-and-compatibility.md)
- [x] Optional stream endpoint (SSE) for browser-native consumers (`GET /api/v1/events`)

Authoritative field list and semantics: [integrations.md](./integrations.md). Policy and compatibility guarantees: [ADR 0001](./adr/0001-events-delivery-and-compatibility.md).

With this ADR accepted and SSE shipped, the current Phase 2 definition is complete.

