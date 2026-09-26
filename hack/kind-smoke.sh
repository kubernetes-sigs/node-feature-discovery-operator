#!/bin/bash
#
# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# kind-smoke.sh deploys the operator with its Helm chart on a throwaway kind
# cluster, applies the sample NodeFeatureDiscovery CR and checks that the
# operand it deploys labels the nodes.
#
# Environment:
#   KIND_CLUSTER      cluster name, must start with "nfd-" (default nfd-op-smoke)
#   KIND_NODE_IMAGE   kind node image (default: kind's default)
#   OPERATOR_IMAGE    operator image to test; built with "make image" when unset
#   OPERAND_IMAGE     expected operand image (default: the sample CR's image)
#   KEEP_CLUSTER      1 keeps the cluster and the kubeconfig after the run
#   SMOKE_TIMEOUT     seconds each wait may take (default 300)
#
# The last check deletes the CR with prunerOnDelete set, so a cluster kept with
# KEEP_CLUSTER=1 no longer runs the operand at the end.
#
# Exit codes: 0 every check passed, 1 a check failed, 2 setup failed.
#
# The script never reads or writes ~/.kube/config: the cluster gets its own
# kubeconfig in a temporary directory.

set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
KIND_CLUSTER=${KIND_CLUSTER:-nfd-op-smoke}
KIND_NODE_IMAGE=${KIND_NODE_IMAGE:-}
OPERATOR_IMAGE=${OPERATOR_IMAGE:-}
SAMPLE_CR=${REPO_ROOT}/config/samples/nfd.kubernetes.io_v1_nodefeaturediscovery.yaml
OPERAND_IMAGE=${OPERAND_IMAGE:-$(awk '$1 == "image:" {print $2; exit}' "${SAMPLE_CR}")}
SMOKE_RULE=${REPO_ROOT}/hack/testdata/smoke-nodefeaturerule.yaml
SMOKE_LABEL=feature.node.kubernetes.io/nfd-operator-smoke
SMOKE_RESOURCE=example.com/nfd-operator-smoke
KEEP_CLUSTER=${KEEP_CLUSTER:-0}
TIMEOUT=${SMOKE_TIMEOUT:-300}
# The sample CR is namespaced here, and the chart watches its release namespace.
NAMESPACE=node-feature-discovery-operator
OPERATOR_DEPLOYMENT=nfd-operator-controller-manager
METRICS_READER_ROLE=nfd-operator-metrics-reader

failures=0
created_cluster=0
port_forward_pid=

log() { echo "[kind-smoke] $*"; }
# setup_fail <message> [log file]: report a setup failure, with the tail of the
# log if one is given (the log is deleted with the work directory on exit).
setup_fail() {
    echo "[kind-smoke] SETUP FAILED: $1" >&2
    if [ -n "${2:-}" ] && [ -f "$2" ]; then
        tail -20 "$2" | sed "s/^/[kind-smoke]   /" >&2
    fi
    exit 2
}
pass() { echo "PASS $1"; }
fail() { echo "FAIL $1: $2"; failures=$((failures + 1)); }

cleanup() {
    rc=$?
    if [ -n "${port_forward_pid}" ]; then
        kill "${port_forward_pid}" 2>/dev/null || true
    fi
    if [ "${created_cluster}" = 1 ] && [ "${KEEP_CLUSTER}" = 1 ]; then
        log "keeping cluster ${KIND_CLUSTER}; kubeconfig: ${KUBECONFIG}"
    else
        if [ "${created_cluster}" = 1 ]; then
            log "deleting cluster ${KIND_CLUSTER}"
            kind delete cluster --name "${KIND_CLUSTER}" --kubeconfig "${KUBECONFIG}" >/dev/null 2>&1 || true
        fi
        rm -rf "${WORKDIR}"
    fi
    exit "${rc}"
}

# wait_until <seconds> <command...>: retry the command every 5 seconds.
wait_until() {
    local deadline=$(( $(date +%s) + $1 ))
    shift
    until "$@" >/dev/null 2>&1; do
        if [ "$(date +%s)" -ge "${deadline}" ]; then
            return 1
        fi
        sleep 5
    done
}

case "${KIND_CLUSTER}" in
    nfd-*) ;;
    *) setup_fail "KIND_CLUSTER=${KIND_CLUSTER} must start with nfd-" ;;
esac
for tool in kind kubectl helm docker jq curl; do
    command -v "${tool}" >/dev/null || setup_fail "${tool} not found"
done
[ -n "${OPERAND_IMAGE}" ] || setup_fail "no operand image in ${SAMPLE_CR}"
clusters=$(kind get clusters 2>/dev/null) || setup_fail "kind get clusters failed"
# Whole-line match in the shell itself: a here-string needs a temporary file
# (bash 3.2), and a pipe into "grep -q" can fail under pipefail.
case $'\n'"${clusters}"$'\n' in
    *$'\n'"${KIND_CLUSTER}"$'\n'*)
        setup_fail "kind cluster ${KIND_CLUSTER} already exists; delete it or set KIND_CLUSTER" ;;
esac

WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/nfd-op-smoke.XXXXXX") || setup_fail "mktemp failed"
export KUBECONFIG=${WORKDIR}/kubeconfig
trap cleanup EXIT

# Operator image: build it unless one is given, and make sure it exists.
if [ -z "${OPERATOR_IMAGE}" ]; then
    OPERATOR_IMAGE=localhost/nfd-operator:smoke
    log "building ${OPERATOR_IMAGE}"
    env -u IMAGE_REGISTRY -u IMAGE_TAG_NAME -u IMAGE_EXTRA_TAG_NAMES \
        make -C "${REPO_ROOT}" image IMAGE_TAG="${OPERATOR_IMAGE}" >"${WORKDIR}/image-build.log" 2>&1 ||
        setup_fail "make image failed" "${WORKDIR}/image-build.log"
fi
case "${OPERATOR_IMAGE}" in
    *:*) ;;
    *) setup_fail "OPERATOR_IMAGE=${OPERATOR_IMAGE} needs an explicit tag" ;;
esac
docker image inspect "${OPERATOR_IMAGE}" >/dev/null 2>&1 || setup_fail "image ${OPERATOR_IMAGE} not in the local docker store"

printf '%s\n' 'kind: Cluster' 'apiVersion: kind.x-k8s.io/v1alpha4' 'nodes:' \
    '  - role: control-plane' '  - role: worker' '  - role: worker' >"${WORKDIR}/kind.yaml" ||
    setup_fail "writing ${WORKDIR}/kind.yaml failed"
log "creating cluster ${KIND_CLUSTER}"
# created_cluster is set only once the create succeeded: a failed create (for
# example because another run created a cluster with the same name in the
# meantime) must never delete a cluster this run does not own. kind removes
# the nodes of a failed create itself.
kind create cluster --name "${KIND_CLUSTER}" --config "${WORKDIR}/kind.yaml" \
    --kubeconfig "${KUBECONFIG}" ${KIND_NODE_IMAGE:+--image "${KIND_NODE_IMAGE}"} \
    >"${WORKDIR}/kind-create.log" 2>&1 || setup_fail "kind create cluster failed" "${WORKDIR}/kind-create.log"
created_cluster=1
context=$(kubectl config current-context 2>/dev/null || true)
[ "${context}" = "kind-${KIND_CLUSTER}" ] || setup_fail "unexpected kube context ${context}"

log "loading ${OPERATOR_IMAGE} and installing the chart"
kind load docker-image "${OPERATOR_IMAGE}" --name "${KIND_CLUSTER}" >/dev/null ||
    setup_fail "kind load docker-image ${OPERATOR_IMAGE} failed"
helm install nfd-operator "${REPO_ROOT}/deploy/helm/nfd-operator" \
    --namespace "${NAMESPACE}" --create-namespace \
    --set image.repository="${OPERATOR_IMAGE%:*}" \
    --set image.tag="${OPERATOR_IMAGE##*:}" \
    --set image.pullPolicy=IfNotPresent \
    >"${WORKDIR}/helm-install.log" 2>&1 || setup_fail "helm install failed" "${WORKDIR}/helm-install.log"
kubectl apply -f "${SAMPLE_CR}" >/dev/null || setup_fail "applying ${SAMPLE_CR} failed"
kubectl -n "${NAMESPACE}" patch nodefeaturediscovery nfd-master-server --type merge \
    -p "{\"spec\":{\"operand\":{\"image\":\"${OPERAND_IMAGE}\"}}}" >/dev/null ||
    setup_fail "setting spec.operand.image failed"

log "checking (each wait up to ${TIMEOUT}s)"

# 1. The operator itself runs.
if kubectl -n "${NAMESPACE}" wait deployment/"${OPERATOR_DEPLOYMENT}" --for=condition=Available --timeout="${TIMEOUT}s" >/dev/null 2>&1; then
    pass "operator-available"
else
    fail "operator-available" "$(kubectl -n "${NAMESPACE}" get pods --no-headers 2>&1 | grep controller-manager | tr -s ' ' | head -3)"
fi

# 2. The operand objects exist and become ready.
for deploy in nfd-master nfd-gc; do
    if wait_until "${TIMEOUT}" kubectl -n "${NAMESPACE}" get deployment "${deploy}" &&
        kubectl -n "${NAMESPACE}" wait deployment/"${deploy}" --for=condition=Available --timeout="${TIMEOUT}s" >/dev/null 2>&1; then
        pass "${deploy}-available"
    else
        fail "${deploy}-available" "$(kubectl -n "${NAMESPACE}" get pods -l app="${deploy}" --no-headers 2>&1 | tr -s ' ' | head -3)"
    fi
done
if wait_until "${TIMEOUT}" kubectl -n "${NAMESPACE}" get daemonset nfd-worker &&
    kubectl -n "${NAMESPACE}" rollout status daemonset/nfd-worker --timeout="${TIMEOUT}s" >/dev/null 2>&1; then
    pass "nfd-worker-rolled-out"
else
    fail "nfd-worker-rolled-out" "$(kubectl -n "${NAMESPACE}" get pods -l app=nfd-worker --no-headers 2>&1 | tr -s ' ' | head -3)"
fi

# 3. Every operand container runs the expected image.
# kubectl exits non-zero when any of the objects is missing, and its partial
# output must not count as "every container runs the operand image".
if raw=$(kubectl -n "${NAMESPACE}" get deployment/nfd-master deployment/nfd-gc daemonset/nfd-worker \
    -o jsonpath='{range .items[*]}{range .spec.template.spec.containers[*]}{.image}{"\n"}{end}{end}' 2>/dev/null); then
    images=$(printf '%s\n' "${raw}" | sed '/^$/d' | sort -u)
else
    images="(cannot read nfd-master, nfd-gc and nfd-worker)"
fi
if [ "${images}" = "${OPERAND_IMAGE}" ]; then
    pass "operand-image"
else
    fail "operand-image" "want ${OPERAND_IMAGE}, got: $(echo "${images}" | tr '\n' ' ')"
fi

nodes=$(kubectl get nodes -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || true)

# nodes_missing <label prefix>: print the nodes without a label with that
# prefix. Fails when the nodes cannot be listed, so that an API error never
# reads as "no node is missing the label".
nodes_missing() {
    local json
    json=$(kubectl get nodes -o json) || return 1
    echo "${json}" | jq -r --arg f "$1" \
        'if (.items | length) == 0 then error("no nodes") else .items[] | select((.metadata.labels | to_entries | map(select(.key | startswith($f))) | length) == 0) | .metadata.name end'
}
all_nodes_labelled() {
    local missing
    missing=$(nodes_missing "$1") || return 1
    [ -z "${missing}" ]
}

# 4. Every node carries feature labels.
if wait_until "${TIMEOUT}" all_nodes_labelled "feature.node.kubernetes.io/"; then
    pass "feature-labels"
else
    fail "feature-labels" "nodes without feature.node.kubernetes.io labels: $(nodes_missing "feature.node.kubernetes.io/" | tr '\n' ' ')"
fi

# 5. Every node has a NodeFeature object.
nodefeatures_complete() {
    local have
    [ -n "${nodes}" ] || return 1
    have=$(kubectl get nodefeatures -A -o jsonpath='{range .items[*]}{.metadata.labels.nfd\.node\.kubernetes\.io/node-name}{"\n"}{end}') || return 1
    for n in ${nodes}; do
        echo "${have}" | grep -qx "${n}" || return 1
    done
}
if wait_until "${TIMEOUT}" nodefeatures_complete; then
    pass "nodefeatures"
else
    fail "nodefeatures" "$(kubectl get nodefeatures -A --no-headers 2>&1 | head -3)"
fi

# 6. A NodeFeatureRule produces its label on every node.
if kubectl apply -f "${SMOKE_RULE}" >/dev/null 2>&1 && wait_until "${TIMEOUT}" all_nodes_labelled "${SMOKE_LABEL}"; then
    pass "nodefeaturerule-label"
else
    fail "nodefeaturerule-label" "nodes without ${SMOKE_LABEL}: $(nodes_missing "${SMOKE_LABEL}" | tr '\n' ' ')"
fi

# check_log <check name> <pattern> <kubectl logs arguments...>: FAIL when the
# log cannot be read, is empty, or matches the extended regexp.
check_log() {
    local name=$1 pattern=$2 out
    shift 2
    out=${WORKDIR}/${name}.log
    if ! kubectl -n "${NAMESPACE}" logs "$@" >"${out}" 2>"${out}.err"; then
        fail "${name}" "cannot read the log: $(head -1 "${out}.err")"
    elif [ ! -s "${out}" ]; then
        fail "${name}" "empty log"
    elif grep -Eq "${pattern}" "${out}"; then
        fail "${name}" "$(grep -E "${pattern}" "${out}" | head -2 | tr '\n' ' ')"
    else
        pass "${name}"
    fi
}

# 7. The operator logged no reconcile errors or panics. controller-runtime
# recovers panics in Reconcile, so a panic does not restart the pod and has
# to be looked for in the log.
check_log operator-log-clean 'Reconciler error|panic' deployment/"${OPERATOR_DEPLOYMENT}" -c manager

# 8. No operand component was denied by RBAC. Some denials do not stop a pod
# from being Ready (nfd-gc serves /healthz before its node informer syncs),
# so they only show up in the logs.
check_log nfd-master-rbac 'forbidden' deployment/nfd-master
check_log nfd-gc-rbac 'forbidden' deployment/nfd-gc
check_log nfd-worker-rbac 'forbidden' -l app=nfd-worker --tail=-1 --prefix

# 9. No operator or operand container restarted. Readiness and the log checks
# only see the current container, so a crash after the first successful start
# would otherwise go unnoticed. A component without any matching pod fails the
# check too, so that a label change cannot turn it into a check of nothing.
restarts=$(kubectl -n "${NAMESPACE}" get pods -o json 2>/dev/null | jq -r '
    [.items[] | {pod: .metadata.name, statuses: (.status.containerStatuses // []),
        component: (if .metadata.labels["control-plane"] == "controller-manager" then "operator" else .metadata.labels.app end)}
        | select(.component == "operator" or .component == "nfd-master" or .component == "nfd-gc" or .component == "nfd-worker")] as $pods
    | ((["operator", "nfd-master", "nfd-gc", "nfd-worker"] - ($pods | map(.component)))[] | "no \(.) pod"),
      ($pods[] | .pod as $p | .statuses[] | select(.restartCount > 0) | "\($p)/\(.name)=\(.restartCount)")') ||
    restarts="(cannot list pods)"
if [ -z "${restarts}" ]; then
    pass "no-restarts"
else
    fail "no-restarts" "$(echo "${restarts}" | tr '\n' ' ')"
fi

# 10. The manager checks every /metrics request itself (TokenReview and
# SubjectAccessReview): no token gets 401, a token without the metrics-reader
# ClusterRole gets 403, a metrics-reader token gets 200 with Prometheus text.
# A 500 means the review calls failed, for example for missing RBAC.
metrics_status() { # metrics_status <local port> [bearer token]: print the HTTP status
    local args=(-sk -o "${WORKDIR}/metrics.out" -w '%{http_code}')
    if [ -n "${2:-}" ]; then
        args+=(-H "Authorization: Bearer $2")
    fi
    curl "${args[@]}" "https://127.0.0.1:$1/metrics" 2>/dev/null || true
}
check_metrics_auth() {
    local reader noperm port no_token no_perm with_role
    if ! kubectl -n "${NAMESPACE}" create serviceaccount smoke-metrics-reader >/dev/null 2>&1 ||
        ! kubectl -n "${NAMESPACE}" create serviceaccount smoke-metrics-noperm >/dev/null 2>&1 ||
        ! kubectl create clusterrolebinding smoke-metrics-reader --clusterrole="${METRICS_READER_ROLE}" \
            --serviceaccount="${NAMESPACE}:smoke-metrics-reader" >/dev/null 2>&1 ||
        ! reader=$(kubectl -n "${NAMESPACE}" create token smoke-metrics-reader 2>/dev/null) ||
        ! noperm=$(kubectl -n "${NAMESPACE}" create token smoke-metrics-noperm 2>/dev/null); then
        fail "metrics-auth" "could not create the test ServiceAccounts, binding or tokens"
        return
    fi
    kubectl -n "${NAMESPACE}" port-forward deployment/"${OPERATOR_DEPLOYMENT}" :8443 >"${WORKDIR}/port-forward.log" 2>&1 &
    port_forward_pid=$!
    if ! wait_until 30 grep -q '^Forwarding from 127.0.0.1:' "${WORKDIR}/port-forward.log"; then
        fail "metrics-auth" "port-forward did not start: $(head -1 "${WORKDIR}/port-forward.log")"
    else
        port=$(sed -n 's/^Forwarding from 127\.0\.0\.1:\([0-9]*\) .*/\1/p' "${WORKDIR}/port-forward.log" | head -1)
        no_token=$(metrics_status "${port}")
        no_perm=$(metrics_status "${port}" "${noperm}")
        with_role=$(metrics_status "${port}" "${reader}")
        if [ "${no_token}/${no_perm}/${with_role}" != 401/403/200 ]; then
            fail "metrics-auth" "want 401/403/200, got ${no_token}/${no_perm}/${with_role}"
        elif ! grep -q '^# HELP' "${WORKDIR}/metrics.out"; then
            fail "metrics-auth" "got 401/403/200, but the 200 response has no '# HELP' line"
        else
            pass "metrics-auth"
        fi
    fi
    kill "${port_forward_pid}" 2>/dev/null || true
    wait "${port_forward_pid}" 2>/dev/null || true
    port_forward_pid=
}
check_metrics_auth

# 11. Deleting the CR with prunerOnDelete set removes what NFD put on the nodes.
# The smoke rule also publishes an extended resource, so the prune Job has to
# patch the nodes' status subresource, which needs its own RBAC rule.
all_nodes_have_resource() {
    local json
    json=$(kubectl get nodes -o json) || return 1
    echo "${json}" | jq -e --arg r "${SMOKE_RESOURCE}" \
        '(.items | length) > 0 and all(.items[]; .status.capacity[$r] != null)' >/dev/null
}
nodes_pruned() {
    local json
    json=$(kubectl get nodes -o json) || return 1
    echo "${json}" | jq -e --arg r "${SMOKE_RESOURCE}" \
        '(.items | length) > 0 and all(.items[]; (.status.capacity[$r] == null) and ((.metadata.labels | keys | map(select(startswith("feature.node.kubernetes.io/"))) | length) == 0))' >/dev/null
}
# cr_gone: only NotFound counts as gone; any other kubectl error does not.
cr_gone() {
    local out
    out=$(kubectl -n "${NAMESPACE}" get nodefeaturediscovery nfd-master-server --ignore-not-found -o name) || return 1
    [ -z "${out}" ]
}
if ! wait_until "${TIMEOUT}" all_nodes_have_resource; then
    fail "prune-on-delete" "extended resource ${SMOKE_RESOURCE} never reached every node"
elif ! kubectl -n "${NAMESPACE}" patch nodefeaturediscovery nfd-master-server --type merge \
    -p '{"spec":{"prunerOnDelete":true}}' >/dev/null 2>&1 ||
    ! kubectl -n "${NAMESPACE}" delete nodefeaturediscovery nfd-master-server --wait=false >/dev/null 2>&1; then
    fail "prune-on-delete" "could not set prunerOnDelete and delete the CR"
elif ! wait_until "${TIMEOUT}" cr_gone; then
    fail "prune-on-delete" "CR not deleted; prune Job: $(kubectl -n "${NAMESPACE}" get job nfd-prune -o jsonpath='{.status}' 2>&1)"
elif ! wait_until "${TIMEOUT}" nodes_pruned; then
    fail "prune-on-delete" "nodes still carry NFD labels or ${SMOKE_RESOURCE}"
else
    pass "prune-on-delete"
fi

if [ "${failures}" -ne 0 ]; then
    log "${failures} check(s) failed"
    exit 1
fi
log "all checks passed"
