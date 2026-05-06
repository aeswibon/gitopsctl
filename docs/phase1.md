# Phase 1 scope

Phase 1 delivers a **trustworthy minimal GitOps loop** that matches the project goal: Git as desired state, external controller, named clusters, CLI + API for day-to-day operations.

## Already implemented (baseline)

- Register/list/status/unregister **applications** and **clusters** (CLI).
- Persist definitions under `configs/applications.json` and `configs/clusters.json`.
- **Controller**: poll Git on an interval, detect new commits, apply YAML manifests via client-go.
- **REST API** (`gitopsctl start`, default `:8080`): CRUD-style routes for apps and clusters, manual app sync, cluster health check endpoint, `/health`.
- Logging (zap), structured API access logs, graceful shutdown on SIGINT/SIGTERM.

## Phase 1 completion checklist (recommended)

These items close the gap between “prototype” and something contributors and users can rely on.

### Quality and safety

1. **Automated tests**: unit tests for validation, config load/save, API handlers (happy paths + errors); targeted tests for Git/K8s boundaries with fakes or integration tags where feasible.
2. **CI**: run `go test ./...`, `go vet ./...`, and `go build` on every PR (e.g. GitHub Actions); pin Go version to `go.mod`.
3. **Version surfacing**: `-v` / `--version` (inject commit/tag at build time via `-ldflags`).
4. **Deterministic bootstrap**: ensure `configs/` creation and empty-file behavior are tested and documented (first-run story).

### Operations

5. **API hardening for real networks**: authentication option (e.g. bearer token or mTLS) **or** clear “bind to localhost only” guidance until auth lands; document threat model in README.
6. **Kubeconfig context**: optional `--context` on cluster registration and apply path so multi-context kubeconfigs behave predictably (today docs previously implied this; behavior should match docs).

### Documentation and UX

7. **Single API reference**: one markdown table or minimal OpenAPI for `/api/v1` (request/response shapes, status codes).
8. **CONTRIBUTING.md**: build, run controller locally, run tests, PR expectations.
9. **CHANGELOG.md** or tagged releases with SemVer once Phase 1 is “done.”

### Explicitly out of scope for Phase 1

- Web UI, Helm/OCI-first workflows, plugin system, webhook-primary sync, advanced policy/notifications (track as Phase 2+).

Use this file as the working definition of “Phase 1 complete” for roadmap and community discussions.
