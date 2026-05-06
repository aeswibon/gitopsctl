# ADR 0001: Events Delivery Guarantees and Compatibility Policy

- Status: Accepted
- Date: 2026-05-06
- Decision owners: GitOpsCTL maintainers
- Related docs: [phase2.md](../phase2.md), [integrations.md](../integrations.md)

## Context

Phase 2 introduces integration events for custom dashboards and automations. External consumers need clear, stable expectations for:

1. Delivery behavior per sink (drop/retry/order)
2. Event schema compatibility over time
3. Upgrade/deprecation policy for event types and fields

Without explicit guarantees, integrators may assume stronger semantics than GitOpsCTL provides and build brittle receivers.

## Decision

### 1) Delivery guarantees by sink

- **JSONL file sink (`--events-file`)**
  - Delivery model: **best-effort append** from process memory to local file.
  - Ordering: process append order.
  - Durability: each record is appended and synced by the sink implementation; records already written remain after process restart.
  - Failure mode: write failures are logged; processing continues.

- **Webhook sink (`--events-webhook`)**
  - Delivery model: **bounded at-least-once attempt semantics**.
  - Retries: transient failures (network errors, `5xx`, `429`) retry with exponential backoff up to configured attempts.
  - Non-retryable errors: most other `4xx` fail fast.
  - Ordering: no global ordering guarantee across all events.
  - Receiver requirement: idempotency by `id` (`X-GitOpsctl-Event-ID`).

- **SSE stream (`GET /api/v1/events`)**
  - Delivery model: **best-effort live stream only**.
  - No replay/persistence guarantee.
  - Slow clients may miss messages due to bounded in-memory buffers.

### 2) Envelope and schema compatibility

- Envelope field `specversion` is currently **`1.0`**.
- For `specversion: "1.0"`:
  - Adding optional fields in `data` is allowed.
  - Reordering JSON object fields is allowed.
  - Existing field names and meanings are not changed.
  - Existing event `type` strings remain stable.

### 3) Breaking changes policy

A change is breaking if it removes/renames fields, changes type semantics incompatibly, or changes stable event type names.

For breaking changes:

1. Introduce a new envelope version (for example `specversion: "2.0"`).
2. Keep old behavior available for a deprecation window of at least **2 minor releases**.
3. Document migration and examples in `docs/integrations.md`.

### 4) Security requirements for webhooks

- Use HTTPS in production.
- Use `--events-webhook-secret` to sign payloads.
- Receivers should verify:
  - HMAC signature (`X-GitOpsctl-Signature`)
  - timestamp freshness (`X-GitOpsctl-Timestamp`)
  - idempotency by event id.

## Consequences

### Positive

- Integrators can safely design resilient consumers (idempotent webhook handlers, tolerant SSE clients).
- Maintainers have a clear compatibility contract for future event evolution.
- Product stance remains CLI-first without coupling to a first-party dashboard.

### Trade-offs

- No strict exactly-once guarantee.
- SSE is not suitable as a durable queue.
- Maintaining compatibility windows increases maintenance overhead for future major changes.

## Future considerations

- Optional dead-letter/replay mechanism for failed webhooks.
- Explicit versioned event-type namespaces for future major revisions.
- Optional persisted event log endpoint for recovery use cases.
