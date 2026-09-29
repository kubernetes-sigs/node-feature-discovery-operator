---
title: "Upgrade"
layout: default
sort: 3
---

# Upgrading NFD-Operator

{: .no_toc}

## Table of contents

{: .no_toc .text-delta}

1. TOC
{:toc}

---

## Operand version

The operator deploys NFD v0.18.0 or newer. It passes nfd-master a `-port`
flag that NFD v0.17.x rejects and that older releases read as the gRPC port.
Set `spec.operand.image` of every
NodeFeatureDiscovery to such a release, for example
`registry.k8s.io/nfd/node-feature-discovery:v0.19.0`.

`spec.resourceLabels` is ignored, because nfd-master removed its
`-resource-labels` flag in NFD v0.17.0. Publish extended resources with a
NodeFeatureRule (`extendedResources`) instead. The operator logs a message when
a NodeFeatureDiscovery still sets the field.

## CRDs

`helm upgrade` does not install or update the CRDs in the chart's `crds/`
directory; Helm applies them only on `helm install`. Apply the new CRDs from a
checkout of the release before you upgrade the chart:

```bash
kubectl apply --server-side --force-conflicts -f deploy/helm/nfd-operator/crds/
```

`--force-conflicts` takes over the fields that Helm set when it installed the
CRDs.

### NodeFeatureRule CRD installed from master

Between April 2024 and this release, the operator's master branch installed
`nodefeaturerules.nfd.k8s-sigs.io` as a namespaced resource (the kustomize files
since April 2024, the Helm chart since June 2024). NFD uses a cluster-scoped
NodeFeatureRule, and the scope of an existing CRD cannot change, so the command
above fails with:

```text
The CustomResourceDefinition "nodefeaturerules.nfd.k8s-sigs.io" is invalid: spec.scope: Invalid value: "Cluster": field is immutable
```

Check the scope:

```bash
kubectl get crd nodefeaturerules.nfd.k8s-sigs.io -o jsonpath='{.spec.scope}'
```

If it prints `Namespaced`, replace the CRD and re-create the rules as
cluster-scoped objects. Deleting the CRD deletes every NodeFeatureRule, so save
them first. Rules with the same name in different namespaces would become one
object: rename them before you start.

1. Save the rules without their namespace and the fields the server sets:

   ```bash
   kubectl get nodefeaturerules.nfd.k8s-sigs.io -A -o json | jq '.items[] |= (del(.metadata.namespace, .metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.managedFields, .metadata.generation) | if .metadata.annotations then .metadata.annotations |= del(.["kubectl.kubernetes.io/last-applied-configuration"]) else . end)' > nodefeaturerules.json
   ```

1. Delete the namespaced CRD, apply the new CRDs and wait for them:

   ```bash
   kubectl delete crd nodefeaturerules.nfd.k8s-sigs.io
   kubectl apply --server-side --force-conflicts -f deploy/helm/nfd-operator/crds/
   kubectl wait --for=condition=Established crd/nodefeaturerules.nfd.k8s-sigs.io
   ```

1. Refresh kubectl's discovery cache. Without this, kubectl still treats
   NodeFeatureRule as namespaced and the next step fails with `NotFound`:

   ```bash
   kubectl api-resources --api-group=nfd.k8s-sigs.io
   ```

1. Re-create the rules:

   ```bash
   kubectl apply -f nodefeaturerules.json
   ```

Then upgrade the chart. An install from the kustomize files (`make deploy`)
needs the same steps; run `make deploy` after them.

## Other changes

- nfd-worker no longer mounts the host's `/usr/src`, like the default of the
  NFD v0.19.0 chart.
- The manager serves its metrics on port 8443 over TLS itself; the
  kube-rbac-proxy sidecar is gone. A scraper needs a token that is allowed to
  GET `/metrics` (the `metrics-reader` ClusterRole) and has to skip certificate
  verification, because the certificate is self-signed.

The [modernization notes](../modernize-notes) list every change.
