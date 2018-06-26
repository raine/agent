---
title: Installation
---

# Installation

The instructions below assume you are running a Kubernetes Cluster version >= 1.3 and are able to run `kubectl` against
your cluster. The following Kubernetes manifests deploy the Timber Agent as a Daemonset, creating a Pod on each of the
Nodes in the Kubernetes Cluster. The Timber Agent will then read and ship logs from `/var/lib/containers`, the default
log location for Docker Containers running on Kubernetes. Configuration for the Timber Agent is stored and read from a
ConfigMap, and the API Key is stored as a Kubernetes Secret.

_Logs collected will only include stdout and stderr from Docker Containers. Configuring containerized applications
is outside the scope of this document._

1. Create Kubernetes Secret with Timber API Key

    ```bash
    kubectl create secret generic timber --from-literal=timber-api-key={{timber_api_key}}
    ```

1. Create Kuberentes Daemonset for Timber Agent

    2a. For Kubernetes Cluster versions >= 1.7

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/timberio/agent/v0.8.3/support/scripts/kubernetes/timber-agent-daemonset.yaml
    ```

    2b. For Kubernetes Cluster versions <= 1.6

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/timberio/agent/v0.8.3/support/scripts/kubernetes/timber-agent-daemonset-legacy.yaml
    ```

## RBAC Support (1.6+ required)

_It is assumed you have enabled and verified RBAC._

The following manifests also create resources in the default namespace, assuming that is the one to be used. If not,
RBAC resources should be created manually.

Create RBAC resources

```bash
kubectl apply -f https://raw.githubusercontent.com/timberio/agent/v0.8.3/support/scripts/kubernetes/timber-agent-daemonset-with-rbac.yaml
```

If you are managing RBAC outside of this install, then you should only need the [Timber Agent ClusterRole].

[Timber Agent ClusterRole]: https://raw.githubusercontent.com/timberio/agent/v0.8.3/support/scripts/kubernetes/timber-agent-clusterrole.yaml
