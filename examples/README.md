# GitOpsCTL Examples

This directory contains examples of how to use GitOpsCTL to manage your Kubernetes applications.

## Quickstart Example: NGINX

To deploy the sample NGINX application in this directory using `gitopsctl`:

1. **Register your local cluster** (e.g. Docker Desktop, OrbStack, Minikube):
   ```bash
   gitopsctl register-cluster -n local-cluster -k ~/.kube/config
   ```

2. **Register the application**, pointing to this repository:
   ```bash
   gitopsctl register-apps \
     -n example-nginx \
     -r https://github.com/aeswibon/gitopsctl.git \
     -p examples/nginx \
     -c local-cluster \
     -i 30s
   ```

3. **Start the controller**:
   ```bash
   gitopsctl start
   ```

The controller will poll the repository and deploy the `deployment.yaml` found in `examples/nginx` to your cluster.
