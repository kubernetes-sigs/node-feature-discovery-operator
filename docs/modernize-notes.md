---
title: "Modernization notes (NFD v0.19.0 operand)"
layout: default
sort: 99
---

# Modernization notes: operand NFD v0.19.0

These notes list every change the operator needed to deploy a working
node-feature-discovery v0.19.0 operand, and the differences from the NFD v0.19.0
Helm chart that were deliberately left in place. The operand objects are still
built by hand in `internal/`. The plan is to replace them with the rendered NFD
Helm chart when the operator moves into the node-feature-discovery repository;
the "Deferred" table is the starting point for mapping the CR fields to chart
values.

Reference: node-feature-discovery v0.19.0, chart
`deployment/helm/node-feature-discovery` rendered with default values (and with
`topologyUpdater.enable=true` for the topology-updater rows).

The ID column refers to the findings of the baseline measurement (B*) and of
the operand review (A4-N*) done for this change.

## Changed

| ID | Object | Field | Before | After | Why |
|---|---|---|---|---|---|
| B11 | nfd-master Deployment | `-port` flag | `--port=12000` unless `spec.operand.servicePort` is set | `--port=8080` unless `spec.operand.servicePort` is set | Since NFD v0.18 `-port` is the single HTTP port for `/metrics` and `/healthz`. With 12000 the process listened away from its probes and was restarted forever. |
| B11 | nfd-master Deployment | `http` containerPort | always 8080 | the same port as `-port` | Keeps the flag, the container port and the probes (which use the port name) in agreement when `servicePort` is set. |
| A4-N1 | nfd-master Deployment | `-resource-labels` | passed when `spec.resourceLabels` is set | never passed; the operator logs that the field is ignored | The flag was removed in NFD v0.17.0 and nfd-master exits on an unknown flag. `spec.resourceLabels` is now ignored; extended resources come from NodeFeatureRule `extendedResources`. |
| A4-N2 | nfd-worker DaemonSet | hostPath of `host-usr-lib`, `host-lib`, `host-proc-swaps` | `/host-usr/lib`, `/host-lib`, `/host-proc/swaps` | `/usr/lib`, `/lib`, `/proc/swaps` | The host side named the in-container paths, which do not exist on the node, so kernel config, builtin modules and swap were never detected. |
| A4-N2 | nfd-worker DaemonSet | `host-usr-src` mount | hostPath `/host-usr/src` | removed | Same path bug. The v0.19.0 chart mounts `/usr/src` only when `worker.mountUsrSrc` is set (default false), because the mount fails on nodes without `/usr/src` and with a read-only `/usr`. |
| none | nfd-worker DaemonSet | `host-sys` mount | read-write | read-only | Same as the v0.19.0 chart. |
| A4-N6 | nfd-gc Deployment, prune Job | image pull policy | always `Always` | `spec.operand.imagePullPolicy`, default `Always` | nfd-master and nfd-worker already followed the CR. |
| B12 | nfd-worker Role | rules | nodefeatures get/create/update | nodefeatures create/get/update/delete; pods get | The worker reads its own pod to set the NodeFeature owner reference and exits when that is forbidden. |
| A4-N3 | nfd-master ClusterRole | NodeFeatureGroup status | kustomize: `nodefeaturegroup/status` (matches nothing); chart: none | `nodefeaturegroups/status` patch/update | NodeFeatureGroup status updates were forbidden. |
| A4-N4 | prune ClusterRole | nodes | nodes | nodes, nodes/status | Pruning removes NFD extended resources through the status subresource; without it the prune Job fails and the CR finalizer is never removed. |
| A4-N5 | topology-updater ClusterRole | rules | pods get (twice) | pods get/list/watch; namespaces get; customresourcedefinitions get/list/watch | The node-scoped pod informer needs list/watch; namespaces and CRDs are read for owner references and the NodeResourceTopology CRD wait. |
| B13 | Chart ClusterRoleBindings for nfd-gc and nfd-topology-updater | subject namespace | `default`, `node-feature-discovery` | the release namespace | The ServiceAccounts live in the release namespace; nfd-gc could not list nodes. |
| B13 | kustomize bindings for nfd-topology-updater, nfd-master, nfd-prune and nfd-worker (`config/rbac/*/`) | subject namespace | `node-feature-discovery` (topology-updater) or `node-feature-discovery-operator` | `default`, like nfd-gc | kustomize rewrites a subject namespace only when it matches the ServiceAccount's own (unset, so `default`) namespace. With `default` every binding follows the namespace set in `config/default`; the stock render is unchanged, and a custom namespace no longer leaves the subjects behind. |
| A4 table C1-C3 | NodeFeature, NodeFeatureRule, NodeFeatureGroup CRDs | schema | copies from NFD v0.12 to v0.14 | NFD v0.19.0 `deployment/base/nfd-crds/nfd-api-crds.yaml` | Ge/Le/GeLe match operators, `type` fields, NodeFeatureGroup `vars`/`varsTemplate`. |
| A4 table C4 | NodeResourceTopology CRD | versions | v1alpha1 | v1alpha1 + v1alpha2 (storage) | nfd-gc and nfd-topology-updater v0.19.0 use v1alpha2. |
| A4-N7 | NodeFeatureDiscovery CRD copies (chart, bundle) | schema | hand-kept, older than several CR fields | the generated `config/crd/bases` file plus the `api-approved.kubernetes.io` annotation (what `kustomize build config/crd` renders) | The chart copy pruned masterEnvs, masterTolerations, workerEnvs, workerTolerations and workerPriorityClassName. The annotation is required for a CRD in a protected `*.kubernetes.io` group. |
| B6 | Operator Deployment (kustomize, chart, both CSVs) | metrics endpoint | kube-rbac-proxy sidecar `gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0` (v0.5.0 in the base CSV) with `--v=10`, in front of the manager on `127.0.0.1:8080` | no sidecar; the manager serves `/metrics` on `:8443` over TLS and checks every request with a TokenReview and a SubjectAccessReview (controller-runtime's `WithAuthenticationAndAuthorization`) | The gcr.io image is gone, so the operator pod never became Available. At `--v=10` the proxy logged its own ServiceAccount token and callers' tokens. The newest kube-rbac-proxy on registry.k8s.io, v0.16.0, has 3 critical and 50 high findings in a trivy scan. The existing proxy ClusterRole already grants the manager's ServiceAccount the two reviews. |
| none | Operator, deletion of a CR with `prunerOnDelete` | prune Job result | an error while the Job has any failed pod, so the finalizer is never removed | done once the Job has a succeeded pod; an error only when the Job's `Failed` condition is `True` | With v0.19.0 the first prune pod can fail and the Job's retry then prunes every node (see Known gaps); the CR hung in deletion. |
| none | Sample CR, CSV alm-examples, manager env, chart values, docs | operand image | `gcr.io/k8s-staging-nfd/node-feature-discovery:master` or `master-minimal` (the bundle CSV example named the operator image) | `registry.k8s.io/nfd/node-feature-discovery:v0.19.0`, pull policy `IfNotPresent` | Release instead of a moving staging tag. `servicePort: 12000` is dropped from the examples. |

## Deferred (differences from the v0.19.0 chart left in place)

None of these makes v0.19.0 fail or mislabel nodes.

| Object | Operator | v0.19.0 chart |
|---|---|---|
| nfd-master | no `-instance` from `spec.instance`; no startup probe; liveness delay 10s; no master config file; tolerates `master` and `control-plane`; seccomp RuntimeDefault; no resources; labels `app: nfd-master`; SA `nfd-master` | `-instance` when set; startup probe; ConfigMap `nfd-master.conf`; `control-plane` only; no seccomp profile; resources; `app.kubernetes.io/*` labels; chart SA name |
| nfd-worker | no container port or probes; extra `source.d` hostPath; `features.d` mount read-write; config key `nfd-worker-conf`; seccomp RuntimeDefault; tolerates all NoSchedule; required linux affinity; no updateStrategy or resources | port 8080 with probes; no `source.d`; `features.d` read-only; key `nfd-worker.conf`; no seccomp profile; no tolerations; maxUnavailable 10%; resources |
| nfd-gc | readiness failureThreshold 10; no resources | resources |
| prune Job | SA `nfd-prune`; extra env `NODE_NAME`; container name `nfd-prune`; tolerates `master` with a preference for control-plane nodes; no resources; no `ttlSecondsAfterFinished` | chart post-delete Job SA and names; resources; TTL 3600 |
| nfd-topology-updater | `-sleep-interval=3s`; no `-watch-namespace` (the default `*` is the same); podresources socket inside the read-only kubelet mount; hostPath `type` Socket/Directory set; extra env `POD_NAME`, `POD_UID`; no `-port` or probes; `allowPrivilegeEscalation: true`; SA `nfd-topology-updater` | 60s; `-watch-namespace=*`; separate podresources mount; no hostPath types; port 8080 with probes; `allowPrivilegeEscalation: false`; SA `<release>-node-feature-discovery-topology-updater` |
| nfd-master ClusterRole | no namespaces watch/list (only used with `restrictions.nodeFeatureNamespaceSelector`, which the CR cannot set); chart lease name `<fullname>-master.nfd.kubernetes.io` (only used with leader election, never enabled) | namespaces watch/list; `nfd-master.nfd.kubernetes.io` |

## Known gaps

- The metrics endpoint uses the self-signed certificate controller-runtime
  generates at start-up, so a scraper has to skip certificate verification
  until a certificate is mounted. The ClusterRole that allows the manager's
  TokenReviews and SubjectAccessReviews keeps its `proxy-role` name.
- The chart ships a second metrics Service
  (`templates/rbac/auth_proxy/service.yaml`) whose selector
  `control-plane: nfd-controller-manager` matches no pod of the chart.
- With NFD v0.19.0 the first `nfd-master -prune` pod can fail on a node with
  "error while patching extended resources: the server rejected our request
  due to an error in our request"; the prune Job's retry then prunes every
  node (2 of 2 kind runs). The cause in nfd-master is not investigated here.
- `make verify` runs the `golangci-lint` found on PATH, as the
  node-feature-discovery Makefile does. CI installs v2.11.4 in
  `scripts/test-infra/verify.sh`; locally, run `make golangci-lint` and put
  `bin/` first on PATH.
- The operator has no default operand image. `NODE_FEATURE_DISCOVERY_IMAGE` is
  set in the manager Deployment and the CSV but is not read by the code, and a
  CR without `spec.operand.image` is rejected by the API server (`image:
  Required value`). The CRD text "[defaults to
  registry.k8s.io/nfd/node-feature-discovery]" is not implemented.
- The CR status reports `Degraded=False` while the operand pods crash.
- `spec.operand.servicePort` has no range validation. It now also sets the
  container port, so an out-of-range value (for example 70000) makes the
  Deployment invalid and the operator logs a reconcile error on every attempt.
- The operand has to be NFD v0.18.0 or newer: nfd-master v0.17.0 has no
  `-port` flag, and in v0.16.0 `-port` is the gRPC port. The operator does
  not check the version of `spec.operand.image`.
- Operator v0.6.0 installs a `nodefeaturerules.nfd.kubernetes.io` CRD that no
  NFD release reads; upgrades leave it behind.
- The operator's master branch installed `nodefeaturerules.nfd.k8s-sigs.io` as
  a namespaced resource from April 2024 on; NFD's is cluster-scoped and the
  scope of a CRD cannot change. `helm upgrade` does not update CRDs either.
  `docs/deployment/upgrade.md` has the steps, tested on kind from a master
  install.
- `make bundle` fails before any of these changes: the ClusterServiceVersions
  carry `version: master` (not semver), `bundle/manifests` holds a stale second
  CSV, and the bundle ships the stale NodeFeatureRule CRD above.
- `config/crd`'s common annotation replaces the NodeResourceTopology CRD's own
  `api-approved.kubernetes.io` value (a link to the KEP) with `unapproved,
  experimental-only` in the kustomize install path.
- The Helm chart runs the operator under the namespace's `default`
  ServiceAccount.
- topology-updater cannot run on kind with Docker Desktop (no NUMA sysfs), so
  its objects were compared with the chart but not run.
