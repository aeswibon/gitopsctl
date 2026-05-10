# GitOpsCTL Examples

This directory contains production-ready examples of GitOpsCTL configurations and manifests.

## Structure

- **`configs/`**: Example `apps.json` and `clusters.json` to bootstrap your setup.
- **`manifests/`**: Sample Kubernetes resources (Deployments, Services, and SOPS-encrypted Secrets).

## How to use these examples

The fastest way to see GitOpsCTL in action is to use these pre-defined examples:

1. **Copy the configs**:
   ```bash
   mkdir -p configs
   cp examples/configs/apps.json configs/apps.json
   cp examples/configs/clusters.json configs/clusters.json
   ```

2. **Configure your cluster**:
   Edit `configs/clusters.json` and ensure the `kubeconfigPath` is correct for your system.

3. **Start GitOpsCTL**:
   ```bash
   gitopsctl start
   ```

4. **Explore the dashboard**:
   ```bash
   gitopsctl dashboard
   ```

For a detailed guide, see [Getting Started](../docs/getting-started.md).
