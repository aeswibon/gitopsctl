# GitOpsCTL codebase overview

GitOpsCTL is a Go CLI and HTTP API that reconciles **Kubernetes manifests in Git** against **registered clusters** (kubeconfig-backed). The controller runs outside the cluster: it polls Git and applies YAML using client-go.

For setup and commands, see the [README](../README.md). For what “Phase 1” includes, see [phase1.md](./phase1.md).

## Layout

```txt
main.go                 → cmd.Execute()
cmd/                    → Cobra commands (apps, clusters, start)
internal/api/           → Echo server, /api/v1 handlers, validator
internal/controller/    → Reconciliation loop, sync triggers, cluster checks
internal/core/app/      → Application model and persistence
internal/core/cluster/  → Cluster model and persistence
internal/core/git/      → Clone/pull/hash
internal/core/k8s/      → Apply manifests
internal/common/        → Shared types (e.g. API errors)
internal/utils/         → CLI list helpers and flags
configs/                → Runtime JSON stores (created when you register)
```

## Request flow (high level)

1. **CLI or API** updates in-memory stores and persists JSON under `configs/`.
2. **`gitopsctl start`** loads apps and clusters, starts the **controller** and **API** goroutines.
3. **Controller** runs per-app reconciliation: fetch Git at `interval`, compare commit hash, apply manifests to the app’s `clusterName` kubeconfig.
4. **Manual sync** sets controller state so the app reconciles immediately; API returns `202 Accepted`.

## Packages worth reading first

| Package | Role |
|---------|------|
| `internal/controller` | Orchestration, timeouts, backoff, goroutine lifecycle |
| `internal/core/app`, `internal/core/cluster` | Source of truth structs and file I/O |
| `internal/api/server.go` | Route wiring and middleware |
| `cmd/start.go` | Process lifecycle (signals, shutdown order) |
