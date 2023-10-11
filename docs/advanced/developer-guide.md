---
title: "Developer guide"
layout: default
sort: 1
---

# Developer guide

{: .no_toc }

## Table of contents

{: .no_toc .text-delta }

1. TOC
{:toc}

## Building the operator

### Download the source code

```bash
git clone https://github.com/kubernetes-sigs/node-feature-discovery-operator
```

### Build the operator image

```bash
IMAGE_REGISTRY=<my registry>
make image
```

#### Push the container image

```bash
IMAGE_REGISTRY=<my registry>
make push
```

Alternatively, instead of specifying variables on the command line,
you can edit the Makefile to permanently change parameter defaults
like name of the image or namespace where the operator is deployed.

## Manual deployment of the operator

After building the image you can simply run

```bash
IMAGE_REGISTRY=<my registry>
make deploy
```

Then create a NodeFeatureDiscovery CR by running

```bash
kubectl apply -f config/samples/nfd.k8s-sigs.io_v1_nodefeaturediscovery.yaml
```

## Undeploy the operator

The operator will use the operand node-feature-discovery
image built from: `https://github.com/kubernetes-sigs/node-feature-discovery`

To uninstall the operator run

```bash
make undeploy
```

## Clean up labels

In case you need to remove the labels created by NFD,
the source Makefile comes with a built in target

```bash
make clean-labels
```

This will clean all labels referencing to
`feature.node.kubernetes.io` and `nfd.node.kubernetes.io`
