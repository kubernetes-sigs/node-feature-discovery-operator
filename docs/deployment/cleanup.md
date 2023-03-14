---
title: "Cleanup"
layout: default
sort: 4
---

# Removing feature labels

From the [Operand repository][nfd] NFD-Master has a special `-prune` command
line flag for removing all nfd-related node labels, annotations and extended
resources from the cluster.

In order to remove all feature labels from the cluster, run the following
command:

```bash
kubectl apply -k https://github.com/kubernetes-sigs/node-feature-discovery/deployment/overlays/prune?ref={{ site.release }}
kubectl -n node-feature-discovery wait job.batch/nfd-master --for=condition=complete && \
    kubectl delete -k https://github.com/kubernetes-sigs/node-feature-discovery/deployment/overlays/prune?ref={{ site.release }}
```

<!-- Links -->
[nfd]: https://github.com/kubernetes-sigs/node-feature-discovery
