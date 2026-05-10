# Getting Started

This guide will help you set up GitOpsCTL to manage your first Kubernetes application using GitOps.

## 1. Prerequisites

- A running Kubernetes cluster (e.g., Minikube, Kind, or OrbStack).
- `kubectl` configured to point to your cluster.
- GitOpsCTL installed (see [Installation](installation.md)).

---

## 2. Quick Demo (Recommended)

If you want to see GitOpsCTL in action immediately, you can use the provided examples.

### 2.1 Prepare Configs
```bash
# Create local configs directory
mkdir -p configs

# Copy example configs
cp examples/configs/apps.json configs/apps.json
cp examples/configs/clusters.json configs/clusters.json
```

> [!IMPORTANT]
> Edit `configs/clusters.json` and ensure the `kubeconfigPath` points to your actual kubeconfig (usually `~/.kube/config`).

### 2.2 Start the Controller
```bash
gitopsctl start
```
GitOpsCTL will begin syncing the `nginx-demo` application from the public examples repository.

### 2.3 Open the Dashboard
In a new terminal:
```bash
gitopsctl dashboard
```

---

## 3. Manual Setup (Step-by-Step)

### 3.1 Register a Cluster
```bash
gitopsctl register-cluster --name local --kubeconfig ~/.kube/config
```

### 3.2 Register an Application
```bash
gitopsctl register-app \
  --name my-app \
  --repo https://github.com/aeswibon/gitops-examples \
  --path manifests/sample-app \
  --cluster local \
  --sync-policy auto
```

### 3.3 Start & Monitor
Start the controller with `gitopsctl start` and use `gitopsctl status-apps` or the `dashboard` to monitor progress.

## Next Steps

- Explore [Configuration](configuration.md) for more advanced options.
- Learn about [Security](features/security.md) including namespace scoping and SOPS.
- Set up [Observability](features/observability.md) with Prometheus and Webhooks.
