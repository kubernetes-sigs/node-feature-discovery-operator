# Node Feature Discovery Operator

The Node Feature Discovery operator manages detection of hardware features and configuration in a kubernetes cluster.
The operator orchestrates all resources needed to run the [Node-Feature-Discovery](https://github.com/kubernetes-sigs/node-feature-discovery) DaemonSet

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack channel](https://kubernetes.slack.com/messages/node-feature-discovery)
- [Mailing list](https://groups.google.com/forum/#!forum/kubernetes-sig-node)


## Building the operator

Checkout the sources

```bash
$ git clone https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

Build the operator image

```bash
make image IMAGE=<my repo>:<my tag>
```

Optionally you can push it to your image repo

```bash
make image-push IMAGE=<my repo>:<my tag>
```

Alternatively, instead of specifying variables on the command line, you can edit the Makefile to permanently change parameter defaults like name of the image or namespace where the operator is deployed.

## Manual deploy of the operator

The default CR will create the operand (NFD) in the `node-feature-discovery-operator` namespace,
the CR can be edited to choose another namespace and image. See the `manifests/0700_cr.yaml` for the default values.

```bash
$ make deploy IMAGE=<my repo>:<my tag>
```

The operator will use the operand node-feature-discovery image built from: https://github.com/kubernetes-sigs/node-feature-discovery

To uninstall the operator run

```bash
$ make undeploy
```

## Extending Node-feature-discovery with sidecar containers and hooks

First see upstream documentation of the hook feature and how to create a correct hook file:

https://github.com/kubernetes-sigs/node-feature-discovery#local-user-specific-features.

The DaemonSet running on the workers will mount the `hostPath: /etc/kubernetes/node-feature-discovery/source.d`.
Additional hooks can than be provided by a sidecar container that is as well running on the workers and mounting the same hostpath and writing the hook executable (shell-script, compiled code, ...) to this directory.

NFD will execute any file in this directory, if one needs any configuration for the hook,
a separate configuration directory can be created under `/etc/kubernetes/node-feature-discovery/source.d`
e.g. `/etc/kubernetes/node-feature-discovery/source.d/own-hook-conf`, NFD will not recurse deeper into the file hierarchy.

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
