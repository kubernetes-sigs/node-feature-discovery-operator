---
title: "Helm"
layout: default
sort: 1
---

# Deployment with Helm

{: .no_toc}

## Table of contents

{: .no_toc .text-delta}

1. TOC
{:toc}

---

Helm chart allow to easily deploy and manage the NFD-operator.

> NOTE: NFD-operator is not ideal for other Helm charts to depend on as that
> may result in multiple parallel NFD-operator deployments in the same cluster
> which is not fully supported by the NFD-operator Helm chart.

## Prerequisites

[Helm package manager](https://helm.sh/) should be installed.

## Deployment

To install the latest stable version:

```bash
export NFD_O_NS=nfd-operator
helm repo add nfd-operator https://kubernetes-sigs.github.io/node-feature-discovery-operator/charts
helm repo update
helm install nfd-operator/nfd-operator --namespace $NFD_O_NS --create-namespace --generate-name
```

To install the latest development version you need to clone the NFD-Operator Git
repository and install from there.

```bash
git clone https://github.com/kubernetes-sigs/node-feature-discovery-operator/
cd node-feature-discovery-operator/deployment/helm
export NFD_O_NS=nfd-operator
helm install nfd-operator ./nfd-operator/ --namespace $NFD_O_NS --create-namespace
```

See the [configuration](#configuration) section below for instructions how to
alter the deployment parameters.

In order to deploy the [minimal](image-variants.md#minimal) image you need to
override the image tag:

```bash
helm install nfd-operator ./nfd-operator/ --set image.tag={{ site.release }}-minimal --namespace $NFD_O_NS --create-namespace
```

## Configuration

You can override values from `values.yaml` and provide a file with custom values:

```bash
export NFD_O_NS=nfd-operator
helm install nfd-operator/nfd-operator -f <path/to/custom/values.yaml> --namespace $NFD_O_NS --create-namespace
```

To specify each parameter separately you can provide them to helm install command:

```bash
export NFD_O_NS=nfd-operator
helm install nfd-operator/nfd-operator --set nameOverride=NFDinstance --namespace $NFD_O_NS --create-namespace
```

## Uninstalling the chart

To uninstall the `nfd-operator` deployment:

```bash
export NFD_O_NS=nfd-operator
helm uninstall nfd-operator --namespace $NFD_O_NS
```

The command removes all the Kubernetes components associated with the chart and
deletes the release.

## Chart parameters

In order to tailor the deployment of the Node Feature Discovery to your cluster needs
We have introduced the following Chart parameters.

### General parameters

| Name | Type | Default | description |
| ---- | ---- | ------- | ----------- |
| `image.repository` | string | `{{ site.container_image | split: ":" | first }}` | NFD image repository |
| `image.tag` | string | `{{ site.release }}` | NFD image tag |
| `image.pullPolicy` | string | `Always` | Image pull policy |
| `imagePullSecrets` | list | [] | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. [More info](https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod) |
| `nameOverride` | string |  | Override the name of the chart |
| `fullnameOverride` | string |  | Override a default fully qualified app name |

### Controller deployment parameters

| Name | Type | Default | description |
| ---- | ---- | ------- | ----------- |
| `controller.image.repository` | string | `{{ site.container_image | split: ":" | first }}` | NFD-Operator image repository |
| `controller.image.tag` | string | `{{ site.release }}` | NFD-Operator image tag |

[rbac]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
