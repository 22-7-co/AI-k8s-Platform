#!/usr/bin/env bash
# L1-B: real PromQL path — host exporter + Prometheus, operator uses PROMETHEUS_MOCK=false.
# Requires an existing kind cluster (e.g. after e2e-kind.sh with KEEP_CLUSTER=true) or any kubectl context.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CLUSTER="${KIND_CLUSTER_NAME:-ai-k8s-e2e}"
QUERY="${PROMETHEUS_QUERY:-gpu_xid_errors_total > 0}"
PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9090}"

log() { echo "==> $*"; }

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "ERROR: kubectl cannot reach a cluster" >&2
  exit 1
fi

if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  kubectl config use-context "kind-${CLUSTER}" 2>/dev/null || true
fi

log "applying training manifests"
kubectl apply -f "${ROOT}/deploy/manifests/operator/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/training/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/serviceaccount.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrole.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrolebinding.yaml"

kubectl delete job training-job -n ai-training --ignore-not-found --wait=true 2>/dev/null || \
  kubectl delete job training-job -n ai-training --ignore-not-found
sleep 2
kubectl apply -f "${ROOT}/deploy/manifests/training/job.yaml"

kubectl wait --for=condition=Ready --timeout=180s -n ai-training \
  pod -l batch.kubernetes.io/job-name=training-job

FAULT_NODE="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
  -o jsonpath='{.items[0].spec.nodeName}')"
log "fault node ${FAULT_NODE}"

log "reset node healing state"
kubectl uncordon "${FAULT_NODE}" 2>/dev/null || true
kubectl taint nodes "${FAULT_NODE}" ai-k8s-platform.io/gpu-fault:NoSchedule- 2>/dev/null || true
kubectl label node "${FAULT_NODE}" ai-k8s-platform.io/healing-state- 2>/dev/null || true
kubectl annotate node "${FAULT_NODE}" ai-k8s-platform.io/healing-completed-at- 2>/dev/null || true

log "starting exporter on :9100"
go run ./cmd/exporter &
EXP_PID=$!
trap 'kill ${EXP_PID} 2>/dev/null || true; "${ROOT}/scripts/prometheus/start-prometheus.sh" down || true' EXIT

for _ in $(seq 1 30); do
  curl -sf "http://127.0.0.1:9100/metrics" >/dev/null 2>&1 && break
  sleep 1
done

log "starting Prometheus (${PROM_URL})"
"${ROOT}/scripts/prometheus/start-prometheus.sh" up
sleep 5
"${ROOT}/scripts/prometheus/start-prometheus.sh" check

log "inject XID for ${FAULT_NODE}"
curl -sf -X POST "http://127.0.0.1:9100/inject/xid?node=${FAULT_NODE}"

log "wait for PromQL instant query to see fault node"
for _ in $(seq 1 30); do
  if curl -sf -G "${PROM_URL}/api/v1/query" --data-urlencode "query=${QUERY}" | grep -q "\"${FAULT_NODE}\""; then
    break
  fi
  sleep 2
done
if ! curl -sf -G "${PROM_URL}/api/v1/query" --data-urlencode "query=${QUERY}" | grep -q "\"${FAULT_NODE}\""; then
  echo "FAIL: PromQL did not return ${FAULT_NODE}" >&2
  curl -sf -G "${PROM_URL}/api/v1/query" --data-urlencode "query=${QUERY}" || true
  exit 1
fi

export PROMETHEUS_MOCK=false
export PROMETHEUS_URL="${PROM_URL}"
export PROMETHEUS_QUERY="${QUERY}"
export HEALING_DRY_RUN=false
export POLL_INTERVAL=8s
export RESCHEDULE_TIMEOUT=120s
export TARGET_NAMESPACES=ai-training
export TRAINING_JOB_NAME=training-job
export TRAINING_JOB_NAMESPACE=ai-training

log "starting operator (real PromQL)"
go run ./cmd/operator &
OP_PID=$!
trap 'kill ${OP_PID} ${EXP_PID} 2>/dev/null || true; "${ROOT}/scripts/prometheus/start-prometheus.sh" down || true' EXIT

for _ in $(seq 1 60); do
  if kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' 2>/dev/null | grep -q true; then
    log "node ${FAULT_NODE} cordoned via real PromQL"
    kill "${OP_PID}" 2>/dev/null || true
    trap - EXIT
    "${ROOT}/scripts/prometheus/start-prometheus.sh" down || true
    kill "${EXP_PID}" 2>/dev/null || true
    log "P3 e2e-promql PASSED (node=${FAULT_NODE}, query=${QUERY})"
    exit 0
  fi
  sleep 2
done

echo "FAIL: node ${FAULT_NODE} not cordoned after real PromQL" >&2
kubectl get nodes -o wide
exit 1
