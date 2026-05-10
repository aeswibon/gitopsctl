# Getting Started

This guide will help you set up GitOpsCTL to manage your first Kubernetes application using GitOps.

## 1. Prerequisites

- A running Kubernetes cluster (e.g., Minikube, Kind, or a cloud provider cluster).
- `kubectl` configured to point to your cluster.
- A Git repository containing Kubernetes manifests (YAML, Kustomize, or Helm).

## 2. Initialize GitOpsCTL

First, ensure you have GitOpsCTL installed (see [Installation](installation.md)).

Create a directory for your GitOpsCTL configuration:

```bash
mkdir my-gitops
cd my-gitops
mkdir configs
```

## 3. Register a Cluster

Register your current Kubernetes cluster:

```bash
gitopsctl register-cluster --name local-cluster --kubeconfig ~/.kube/config
```

This creates `configs/clusters.json`.

## 4. Register an Application

Register an application by pointing to a Git repository:

```bash
gitopsctl register-app \
  --name sample-app \
  --repo https://github.com/aeswibon/gitops-examples \
  --branch main \
  --path manifests/sample-app \
  --cluster local-cluster \
  --sync-policy auto
```

This creates `configs/apps.json`.

## 5. Start the Controller

Now, start the GitOpsCTL controller:

```bash
gitopsctl start
```

You should see logs indicating that GitOpsCTL is cloning the repository and applying manifests.

## 6. Open the Dashboard (TUI)

In a new terminal window, open the interactive dashboard:

```bash
gitopsctl dashboard
```

From here, you can:
- View the status of all registered applications and clusters.
- Manually trigger synchronizations.
- Approve pending commits (if using `manual` sync policy).
- View live activity logs.

## 7. Make a Change

1. Push a change to your Git repository (e.g., update the number of replicas in a Deployment).
2. Watch GitOpsCTL automatically detect the change and apply it to your cluster.
3. Check the dashboard to see the updated synchronization status and resource health.

## Next Steps

- Explore [Configuration](configuration.md) for more advanced options.
- Learn about [Security](features/security.md) including namespace scoping and SOPS.
- Set up [Observability](features/observability.md) with Prometheus and Webhooks.
