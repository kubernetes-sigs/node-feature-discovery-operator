---
title: "Quick start"
layout: default
sort: 2
---

# Quick start

Get the source code

```bash
git clone https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

Deploy the operator

```bash
IMAGE_TAG=k8s.gcr.io/nfd/node-feature-discovery-operator:{{ site.operator_version }}
make deploy
```

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
  "beta.kubernetes.io/arch": "amd64",
  "beta.kubernetes.io/os": "linux",
  "feature.node.kubernetes.io/cpu-cpuid.ADX": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AESNI": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AVX": "true",
...
```
