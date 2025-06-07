# GitOpsCTL: A Lightweight GitOps Control Plane for Kubernetes

**GitOpsCTL** (GitOps Control Tool) is a minimalistic, self-hosted, and externally managed GitOps controller written in Go. Designed to complement existing tools like ArgoCD and FluxCD, GitOpsCTL offers a simpler, more flexible alternative for Kubernetes application deployments, especially suited for smaller teams, edge environments, or scenarios requiring fine-grained external control.

## Table of Contents

- [🚀 Why GitOpsCTL?](#why-gitopsctl)
- [✨ Features (Phase 1)](#features-phase-1)
- [🏗️ Architecture Goals](#architecture-goals)
- [🏁 Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Clone the Repository](#clone-the-repository)
  - [Install Dependencies & Build](#install-dependencies--build)
- [📖 Usage](#usage)
  - [Register an Application](#register-an-application)
  - [Check Application Status](#check-application-status)
  - [Start the Controller](#start-the-controller)
  - [Example Workflow](#example-workflow)
- [⚙️ Configuration](#configuration)
- [📂 Project Structure (Phase 1)](#project-structure-phase-1)
- [➡️ Next Steps (Future Phases)](#next-steps-future-phases)
- [🤝 Contributing](#contributing)
- [📄 License](#license)

## Why GitOpsCTL?

Traditional GitOps tools are powerful but can be resource-intensive, opinionated, or tightly coupled to the cluster they manage. GitOpsCTL addresses these concerns by being:

- **Lightweight**: Built with Go for efficiency and minimal overhead.
- **External**: Manages deployments from outside your Kubernetes cluster(s), providing a single control plane for multiple environments.
- **GitOps-Driven**: Continuously watches Git repositories for desired state and applies changes to target clusters.
- **Complementary**: Provides a simpler reconciliation loop, allowing you to build custom deployment logic on top of a solid GitOps foundation.

## Features (Phase 1)

This initial phase focuses on the core reconciliation loop:

- **CLI for App Registration**: Easily define new applications with Git repository URLs, manifest paths, and target Kubernetes clusters via command-line.
- **Git Polling**: Periodically checks registered Git repositories for changes to your application manifests.
- **Kubernetes Manifest Sync**: Automatically applies Kubernetes YAML manifests to your target cluster(s) using client-go when changes are detected in Git.
- **Single Kubeconfig Support**: Connects to a Kubernetes cluster using a specified kubeconfig file (works seamlessly with local setups like OrbStack for Mac users).
- **Basic Logging & Status**: Provides console logging for operations and a CLI command to inspect the current sync status of registered applications.

## Architecture Goals

GitOpsCTL is built with a clear architectural vision:

- **External Control Plane**: Operates outside the Kubernetes cluster, offering a broader view and management capabilities.
- **Reconciler Pattern**: Continuously aligns the actual state of your applications in Kubernetes with the desired state defined in Git.
- **Modular Design**: Components like the Git watcher, sync engine, and API (future) are loosely coupled for extensibility and maintainability.
- **Go-Native**: Leverages Go's concurrency model and client-go for efficient Kubernetes interactions.

## Getting Started

### Prerequisites

- **Go (1.20+)**: Install Go on your system.
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

### Register an Application

Define your GitOps application by specifying its Git repository, the path to its Kubernetes manifests, the target kubeconfig file, and the polling interval.

```bash
./gitopsctl register \
  --name my-nginx-app \
  --repo https://github.com/your-github-user/your-gitops-repo.git \
  --path k8s/manifests/nginx \
  --kubeconfig ~/.kube/config \
  --interval 30s
```

- `--name`: A unique identifier for your application.
- `--repo`: The URL of your Git repository (HTTPS or SSH).
- `--path`: The subdirectory within your repository containing Kubernetes YAML files.
- `--kubeconfig`: The path to your Kubernetes kubeconfig file. For OrbStack, `~/.kube/config` usually works.
- `--interval`: How often GitOpsCTL should poll the Git repository for changes (e.g., 30s, 5m, 1h).

After registration, an `applications.json` file will be created/updated in the `configs/` directory, storing your application definitions.

### Check Application Status

You can inspect the current state of all registered applications:

```bash
./gitopsctl status
```

This will show details like the application name, Git repository, current status, and the last synced Git commit hash.

### Start the Controller

Run the main controller to begin the GitOps reconciliation loop:

```bash
./gitopsctl start
```

The controller will start polling your registered Git repositories, applying any detected changes to your Kubernetes cluster. You'll see logs in your terminal indicating its activity.

To stop the controller, simply press `Ctrl+C`. It will perform a graceful shutdown.

### Example Workflow

1. **Register**: Register an application as shown above.
2. **Start**: Run `./gitopsctl start`. Observe the initial deployment of your manifests to Kubernetes. Verify with `kubectl get all -n <your-namespace>`.
3. **Modify**: Make a change to a Kubernetes manifest file in your Git repository (e.g., change an image tag, increase replica count).
4. **Commit & Push**: Commit your changes and push them to your remote Git repository.
5. **Observe**: Within the specified `--interval`, GitOpsCTL will detect the change, pull the new version, and apply the updated manifests to your Kubernetes cluster. You'll see corresponding logs, and `kubectl get all -n <your-namespace>` will reflect the changes.

## Configuration

Application definitions are stored in `configs/applications.json`. You can manually inspect or edit this file, but it's recommended to use the `gitopsctl register` command for consistency.

```json
[
  {
    "name": "my-nginx-app",
    "repoURL": "https://github.com/your-github-user/your-gitops-repo.git",
    "path": "k8s/manifests/nginx",
    "kubeconfigPath": "/Users/youruser/.kube/config",
    "interval": "30s",
    "lastSyncedGitHash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
    "status": "Synced",
    "message": "Successfully synced to a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"
  }
]
```

## Project Structure (Phase 1)

```txt
gitopsctl/
├── main.go               # Main entry point
├── cmd/                  # Cobra CLI commands
│   ├── root.go           # Root command setup
│   ├── register.go       # Register application command
│   ├── start.go          # Start controller command
│   └── status.go         # Show application status command
├── internal/
│   ├── app/              # Application definition and persistence logic
│   │   ├── app.go
│   ├── git/              # Git operations (clone, pull, hash tracking)
│   │   ├── git.go
│   ├── k8s/              # Kubernetes client-go operations (apply manifests)
│   │   ├── k8s.go
│   └── controller/       # Core reconciliation logic
│       ├── controller.go
│       └── types.go
└── configs/              # Directory for application definitions
    └── applications.json # Stores registered app data
```

## Next Steps (Future Phases)

This project is planned for phased development. Here's a glimpse of what's coming:

### Phase 2: API & Multi-cluster

- REST API for managing applications programmatically.
- Support for multiple Kubernetes clusters from a single controller instance.
- Optional webhook triggers for faster Git event detection.

### Phase 3: UI, Extensibility, and Plugins

- A minimal web UI dashboard for visual monitoring.
- Advanced sync strategies (manual approval, scheduled syncs).
- Plugin interface for Helm, OCI, and custom templating engines.
- Integration with notification systems.

## Contributing

We welcome contributions! If you have ideas, bug reports, or want to contribute code, please feel free to open issues or pull requests.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
