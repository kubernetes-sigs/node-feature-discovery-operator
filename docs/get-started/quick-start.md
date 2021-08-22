---
title: "Quick start"
layout: default
sort: 2
---

# Requirements

1. Linux (x86_64/Arm64/Arm)
1. [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl)
   (properly set up and configured to work with your Kubernetes cluster)

# Quick start

Get the source code

```bash
git clone -b {{ site.release }} https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

Deploy the operator

```bash
IMAGE_TAG={{ site.container_image }}
make deploy
```

Create a NodeFeatureDiscovery instance

```bash
kubectl apply -f config/samples/nfd.kubernetes.io_v1_nodefeaturediscovery.yaml
```

## Image variants

Node-Feautre-Discovery-Operator currently offers two variants
of the container image. The "full" variant is currently
deployed by default.

### Full

This image is based on
[debian:buster-slim](https://hub.docker.com/_/debian) and contains a full Linux
system for doing live debugging and diagnosis of the operator.

### Minimal

This is a minimal image based on
[gcr.io/distroless/base](https://github.com/GoogleContainerTools/distroless/blob/master/base/README.md)
and only supports running statically linked binaries.

The container image tag has suffix `-minimal`
(e.g. `{{ site.container_image }}-minimal`)

## Verify

The Operator will deploy NFD based on the information
on the NodeFeatureDiscovery CR instance,
after a moment you should be able to see

```bash
$ kubectl -n node-feature-discovery-operator get ds,deploy
NAME                        DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/nfd-worker   3         3         3       3            3           <none>          5s
NAME                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nfd-master   1/1     1            1           17s
```

Check that NFD feature labels have been created

```bash
$ kubectl get no -o json | jq .items[].metadata.labels
{
  "beta.kubernetes.io/arch": "amd64",
  "beta.kubernetes.io/os": "linux",
  "feature.node.kubernetes.io/cpu-cpuid.ADX": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AESNI": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AVX": "true",
...
```
