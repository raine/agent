---
Title: Troubleshooting
---

# Troubleshooting

## Pods fail to start

You can verify the pods are running with kubectl.

```bash
kubectl get pods -l name=timber-agent
```

```text
NAME                 READY     STATUS    RESTARTS   AGE
timber-agent-gxb47   2/2       Running   0          1m
```

If the pods are not running or initializing, you can view more details about their failures with kubectl.

```bash
kubectl describe pods -l name=timber-agent
```

This information should help you understand why the integration is not starting in your cluster. Errors with the cluster configuration are outside the scope of this documentation, and you should refer to the official Kubernetes documentation and community for further assistance.

## Logs are not showing up in the UI

If logs are not showing up in the UI, check that the pods are running. If running, you can view the agent logs via

```bash
kubectl logs -l name=timber-agent -c timber-agent
```

Check this output for warning or error messages.

Additionally, your file may have matched an exclusion filter. If this is the case, there will be a log entry similar to the following to inform you:

```text
File logs will not be forwarded due to matching an exclusion filter: FILE_PATH MATCHING_FILTER
```

Additionally, your file may have matched an exclusion filter. Info logs are printed with `File logs will not be forwarded due to matching an exclusion filter` with which file was excluded and what filter matched.

## Failed to read metadata

The integration has a soft dependency on the Kubernetes API in order to gather facts about each log's source. If the
Kubernetes API is unavailable when a request is made, the integration will retry the request (using an exponential
backoff) until the service is available again.

The integration will continue to forward logs to the Timber service even when the Kubernetes API is unavailable.
These logs, though, will not have additional metadata about the cluster, and you will see "Failed to retrieve..." in
the agent logs.

If the Kubernetes API is unavailable for a long period of time (on the order of 5 to 10 minutes) or is unavailable
from the start, it means there is an issue with the kubectl proxy setup or the node itself.

The kubectl proxy logs should indicate if an error has occurred. If the logs do not show an error, verify that the
`TIMBER_AGENT_PROXY_SERVICE_HOST` and `TIMBER_AGENT_PROXY_SERVICE_PORT` values for the Timber Agent match the proxy
configuration.

If these evironment variables values are correct, RBAC permissions could be causing the problem. RBAC permissions can
be verified by launching a Pod with the same RBAC sevice account. This Pod should a long running I am container to
perform and validate requests against the Kubernetes API.

## Tectonic - Failed to mount /etc/hostname

Some users have reported an error when attempting to run the Timber Agent on a Tectonic based installation of Kubernetes. The issue is that `/etc/hostname` is not a file on CoreOS workers launched by Tectonic and therefore cannot be mounted into the Timber Agent container. Below are kubectl commands applying manifests that have been changed to work with Tectonic:

### Kubernetes >= 1.7

```bash
kubectl apply -f https://raw.githubusercontent.com/timberio/agent/master/support/scripts/kubernetes/tectonic-timber-agent-daemonset.yaml
```

### Kubernetes with RBAC >= 1.7

```bash
kubectl apply -f https://raw.githubusercontent.com/timberio/agent/master/support/scripts/kubernetes/tectonic-timber-agent-daemonset-rbac.yaml
```

### Kubernetes <= 1.6

```bash
kubectl apply -f https://raw.githubusercontent.com/timberio/agent/master/support/scripts/kubernetes/tectonic-timber-agent-daemonset-legacy.yaml
```

The only changes in these manifests are the removal of mounting `/etc/hostname/`, the source of the problem. By default, `/etc/hostname` is mounted inside of the Timber Agent container to add a container's host to its log metadata.

_Note: Logs shipped by agents launched from these manifests will have their hostnames set to the container name of the Timber Agent._
