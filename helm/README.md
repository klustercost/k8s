this deploys the kubecost in an existing cluster

## Install from OCI registry

```bash
helm install klustercost oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0
```

To pull without installing:

```bash
helm pull oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0
```