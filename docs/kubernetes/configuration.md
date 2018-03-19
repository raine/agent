---
Title: Configuration
---

# Configuration

Timber’s Kubernetes integration options are supplemental to the Timber Agent configuration, under the `[kubernetes]` configuration section.

Configuration Options

- `exclude`

    The exclude table is used to specify log sources that should be ignored. Key value pairs are of the form:

    `field = "filter_string"`

    Here `field` is an identifying piece of Kubernetes metadata and `filter_string` is a comma seprated list of regex expressions to match against.

    The following Kubernetes metadata fields are filterable:
        - deployments
        - namespaces
        - pods

    Defaults to:

    ```toml
    [kubernetes.exclude]
    namespaces = "kube-system"
    pods = "timber-agent"
    ```
