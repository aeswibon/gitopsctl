# GitOpsCTL: A Lightweight GitOps Control Plane for Kubernetes

**GitOpsCTL** (GitOps Control Tool) is a minimalistic, self-hosted, and externally managed GitOps controller written in Go. Designed to complement existing tools like ArgoCD and FluxCD, GitOpsCTL offers a simpler, more flexible alternative for Kubernetes application deployments, especially suited for smaller teams, edge environments, or scenarios requiring fine-grained external control.

## Goals

The project exists to provide a **small, explicit GitOps loop**: desired state lives in Git; GitOpsCTL **watches that Git**, **applies Kubernetes manifests** to **named clusters**, and exposes **CLI and HTTP API** so people and automation can register workloads, trigger syncs, and inspect status—without requiring a full in-cluster control plane such as Argo CD or Flux.

In one sentence: **GitOpsCTL keeps Kubernetes aligned with Git using a minimal external controller.**

Everything beyond that loop (rich UI, plugins, webhooks-first workflows, advanced policy) is optional evolution **after** reconciliation and operations are reliable and well documented.

## Who this is for

- **Platform and DevOps engineers** who want Git-as-source-of-truth deploys with a thin controller they can run beside their existing toolchain.
- **SREs and on-call** who need logs, status, and a way to confirm what revision synced—or to kick a sync without digging through cluster internals only.
- **Small teams, edge, or local Kubernetes setups** where a lightweight external reconciler is easier to own than a large GitOps stack in-cluster.
- **Automation authors** integrating registration, sync, or health checks via the REST API (`/api/v1`) from pipelines or internal tools.

### Who this is not for (today)

- Teams that need **first-class Helm/OCI charts**, **DR admission hooks**, or **deep multi-tenant RBAC on the control plane** out of the box—those may land later; compare with mature GitOps products if that is your baseline.
- Organizations that require **only Git webhook-driven sync** with no polling path as a supported primary mode (polling is the core today; webhooks are a planned enhancement).

## Table of Contents

- [🎯 Goals](#goals)
- [👥 Who this is for](#who-this-is-for)
- [🚀 Why GitOpsCTL?](#why-gitopsctl)
- [✨ Features (Phase 1)](#features-phase-1)
- [🏗️ Architecture Goals](#architecture-goals)
- [🏁 Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Clone the Repository](#clone-the-repository)
  - [Install Dependencies & Build](#install-dependencies--build)
- [📖 Usage](#usage)
  - [Register a cluster](#register-a-cluster)
  - [Register an application](#register-an-application)
  - [Check application status](#check-application-status)
  - [Start the controller](#start-the-controller)
  - [Example workflow](#example-workflow)
- [⚙️ Configuration](#configuration)
- [📂 Project structure](#project-structure)
- [➡️ Next steps (future phases)](#next-steps-future-phases)
- [🤝 Contributing](#contributing)
- [📄 License](#license)

## Why GitOpsCTL?

Traditional GitOps tools are powerful but can be resource-intensive, opinionated, or tightly coupled to the cluster they manage. GitOpsCTL addresses these concerns by being:

- **Lightweight**: Built with Go for efficiency and minimal overhead.
- **External**: Manages deployments from outside your Kubernetes cluster(s), providing a single control plane for multiple environments.
- **GitOps-Driven**: Continuously watches Git repositories for desired state and applies changes to target clusters.
- **Complementary**: Provides a simpler reconciliation loop, allowing you to build custom deployment logic on top of a solid GitOps foundation.

## Features (Phase 1)

A concrete **Phase 1 checklist** (what is done vs still recommended before calling Phase 1 “complete”) lives in [docs/phase1.md](docs/phase1.md).

This phase focuses on the core reconciliation loop and operational APIs:

- **CLI for apps and clusters**: Register applications (Git URL, manifest path, poll interval, target cluster) and register multiple Kubernetes clusters (kubeconfig-backed) via command-line subcommands.
- **Git polling**: Periodically checks registered Git repositories for manifest changes.
- **Kubernetes manifest sync**: Applies YAML manifests to target cluster(s) with client-go when Git moves ahead.
- **REST API**: Manage applications and clusters and trigger sync or cluster checks over HTTP (`gitopsctl start` serves `/api/v1` by default on `:8080`; use `--api-address` to change the bind address).
- **Logging and status**: Structured logs and CLI commands to inspect registration and sync status.

## Architecture Goals

GitOpsCTL is built with a clear architectural vision:

- **External Control Plane**: Operates outside the Kubernetes cluster, offering a broader view and management capabilities.
- **Reconciler Pattern**: Continuously aligns the actual state of your applications in Kubernetes with the desired state defined in Git.
- **Modular design**: Git operations, Kubernetes apply, reconciliation, and HTTP API are separated so each can evolve without collapsing into one blob.
- **Go-Native**: Leverages Go's concurrency model and client-go for efficient Kubernetes interactions.

## Getting Started

### Prerequisites

- **Go (1.24+)**: Match `go.mod`; install Go on your system.
- **Git**: Ensure Git is installed and configured on your machine.
- **Kubernetes Cluster**: A running Kubernetes cluster.
  - **For Mac users**: We highly recommend OrbStack for a fast and lightweight local Kubernetes environment. Enable Kubernetes in OrbStack's settings.
  - Ensure your kubectl is configured to connect to your cluster (e.g., via ~/.kube/config).

### Clone the Repository

```bash
git clone https://github.com/aeswibon/gitopsctl.git
cd gitopsctl
```

### Install Dependencies & Build

```bash
go mod tidy
go build -o gitopsctl .
```

This will create an executable binary named gitopsctl in your current directory.

## Usage

### Register a cluster

Applications deploy to a **named cluster** that must exist in `configs/clusters.json`. Register one first (example uses your default kubeconfig):

```bash
./gitopsctl register-cluster \
  --name production \
  --kubeconfig ~/.kube/config
```

Short flags: `-n` for name, `-k` for kubeconfig. Optional: `--context`, `--test` to verify connectivity, `--dry-run`, `--force`.

Clusters are stored in `configs/clusters.json`.

### Register an application

Point at a Git repo, manifest path **within that repo**, **cluster name** (must match a registered cluster), and poll interval:

```bash
./gitopsctl register-apps \
  --name my-nginx-app \
  --repo https://github.com/your-github-user/your-gitops-repo.git \
  --path k8s/manifests/nginx \
  --cluster production \
  --interval 30s
```

Short flags: `-n` name, `-r` repo, `-p` path, `-c` cluster, `-i` interval. Optional: `-b`/`--branch` (default `main`), `--dry-run`, `--force`.

After registration, `configs/applications.json` is created or updated.

### Check application status

Inspect registered applications (status, last synced commit, messages):

```bash
./gitopsctl status-apps
```

Use flags such as `--output json`, `--details`, or `--sort-by name` for different views.

### Start the controller

Run the main controller to begin the GitOps reconciliation loop:

```bash
./gitopsctl start
```

The controller starts polling registered Git repositories and applying changes to your clusters. An HTTP API is started alongside it (default listen address `:8080`; override with `--api-address`, for example `--api-address 127.0.0.1:9090`). You'll see logs in your terminal indicating activity.

To stop the controller, press `Ctrl+C`. It performs a graceful shutdown (including the API server).

### Example workflow

1. **Register cluster**: `./gitopsctl register-cluster -n production -k ~/.kube/config` (add `--test` if you want a connectivity check).
2. **Register application**: `./gitopsctl register-apps -n my-nginx-app -r <repo> -p k8s/manifests/nginx -c production -i 30s`.
3. **Start**: Run `./gitopsctl start`. Observe the initial deployment of your manifests to Kubernetes. Verify with `kubectl get all -n <your-namespace>`.
4. **Modify**: Change a manifest in Git (for example image tag or replicas).
5. **Commit and push**: Push to the branch your app tracks (default `main` unless you set `-b`).
6. **Observe**: Within the poll `--interval`, GitOpsCTL detects the update, pulls, and applies. Confirm with `./gitopsctl status-apps` and `kubectl`.

## Configuration

Application definitions are stored in `configs/applications.json`. Cluster registrations are stored in `configs/clusters.json`. You can inspect or edit these files manually, but using the CLI (or API) keeps shape and validation consistent.

```json
[
  {
    "name": "my-nginx-app",
    "repoURL": "https://github.com/your-github-user/your-gitops-repo.git",
    "branch": "main",
    "path": "k8s/manifests/nginx",
    "clusterName": "production",
    "interval": "30s",
    "lastSyncedGitHash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
    "status": "Synced",
    "message": "Successfully synced to a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"
  }
]
```

## Project structure

```txt
gitopsctl/
├── main.go               # Entry: delegates to cmd
├── cmd/                  # Cobra CLI (app/cluster register, list, status, start, …)
├── internal/
│   ├── api/              # Echo HTTP server and /api/v1 handlers
│   ├── controller/       # Reconciliation loop and controller commands
│   ├── core/             # Domains: app, cluster, git, k8s (load/save, integrations)
│   ├── common/           # Shared types and validation helpers
│   └── utils/            # CLI helpers (flags, list runners, …)
└── configs/              # Created at runtime
    ├── applications.json # Registered applications
    └── clusters.json     # Registered clusters
```

## Next steps (future phases)

Development is phased; some items below already exist in code.

### API and multi-cluster (partially delivered)

- REST API for apps and clusters is available under `/api/v1` when the controller is running.
- Multiple kubeconfig-backed clusters are supported from one controller process.
- **Still planned**: optional webhook-driven sync as a first-class complement to polling.

### Phase 3: UI, extensibility, and plugins

- A minimal web UI dashboard for visual monitoring.
- Advanced sync strategies (manual approval, scheduled syncs).
- Plugin interface for Helm, OCI, and custom templating engines.
- Integration with notification systems.

## Contributing

We welcome contributions! If you have ideas, bug reports, or want to contribute code, please feel free to open issues or pull requests.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
