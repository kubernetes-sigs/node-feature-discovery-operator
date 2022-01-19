---
title: "Quick start"
layout: default
sort: 2
---

# Quick start

Get the source code

```bash
git clone -b {{ site.release }} https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

Deploy the operator. This step also creates a sample `NodeFeatureDiscovery`
object deploying the operand in default configuration.

```bash
IMAGE_TAG={{ site.container_image }}
make deploy
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
  "beta.kubernetes.io/arch": "amd64",
  "beta.kubernetes.io/os": "linux",
  "feature.node.kubernetes.io/cpu-cpuid.ADX": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AESNI": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AVX": "true",
...
```
