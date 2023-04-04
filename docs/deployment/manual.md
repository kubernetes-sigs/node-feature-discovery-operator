---
title: "Manual deployment"
layout: default
sort: 2
---

# Manual deployment

{: .no_toc}

## Table of contents

{: .no_toc .text-delta}

1. TOC
{:toc}

---
# Requirements

1. Linux (x86_64/Arm64/Arm)
1. [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl)
   (properly set up and configured to work with your Kubernetes cluster)

# Manual deployment

Get the source code

```bash
git clone -b {{ site.release }} https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

Deploy the operator

> You can use the `IMAGE_TAG` environment variable to specify the container
> image to use.

```bash
IMAGE_TAG={{ site.container_image }}
make deploy
```

By default the operator will watch `NodeFeatureDiscovery` objects
only in the namespace where the operator is deployed in. This is
specified by the `WATCH_NAMESPACE` env variable in the operator
deployment manifest. If unset the operator will watch ALL
namespaces.

Create a NodeFeatureDiscovery instance

```bash
kubectl apply -f config/samples/nfd.kubernetes.io_v1_nodefeaturediscovery.yaml
```

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
  "feature.node.kubernetes.io/cpu-cpuid.ADX": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AESNI": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AVX": "true",
  "kubernetes.io/arch": "amd64",
  "kubernetes.io/os": "linux",
...
```

# Uninstallation

If you followed the deployment instructions from the above you
can simply do:

```bash
kubectl -n nfd-operator delete NodeFeatureDiscovery my-nfd-deployment
```

Optionally, you can also remove the namespace:

```bash
kubectl delete ns nfd-operator
```

See the [node-feature-discovery-operator][nfd-operator] and [OLM][OLM] project
documentation for instructions for uninstalling the operator and operator
lifecycle manager, respectively.

<!-- Links -->
[nfd-operator]: https://github.com/kubernetes-sigs/node-feature-discovery-operator
[OLM]: https://github.com/operator-framework/operator-lifecycle-manager
