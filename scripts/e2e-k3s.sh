#!/usr/bin/env bash
# P2 e2e on existing k3s/kubectl context: operator runs locally, mock PromQL fault.
# For strict 2-node L1, use ./scripts/e2e-kind.sh when kind image pull works.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { echo "==> $*"; }

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "ERROR: kubectl cannot reach a cluster" >&2
  exit 1
fi

log "applying manifests"
kubectl apply -f "${ROOT}/deploy/manifests/operator/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/training/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/serviceaccount.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrole.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrolebinding.yaml"

log "cleaning previous training job (repeatable e2e)"
kubectl delete job training-job -n ai-training --ignore-not-found --wait=true 2>/dev/null || \
  kubectl delete job training-job -n ai-training --ignore-not-found
sleep 2

kubectl apply -f "${ROOT}/deploy/manifests/training/job.yaml"

log "waiting for training pod"
kubectl wait --for=condition=Ready --timeout=180s -n ai-training \
  pod -l batch.kubernetes.io/job-name=training-job

FAULT_NODE="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
  -o jsonpath='{.items[0].spec.nodeName}')"
OLD_POD="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
  -o jsonpath='{.items[0].metadata.name}')"
log "training pod ${OLD_POD} on ${FAULT_NODE}"

log "reset node ${FAULT_NODE} healing labels/taints"
kubectl uncordon "${FAULT_NODE}" 2>/dev/null || true
kubectl taint nodes "${FAULT_NODE}" ai-k8s-platform.io/gpu-fault:NoSchedule- 2>/dev/null || true
kubectl label node "${FAULT_NODE}" ai-k8s-platform.io/healing-state- 2>/dev/null || true
kubectl annotate node "${FAULT_NODE}" ai-k8s-platform.io/healing-completed-at- 2>/dev/null || true

NODE_COUNT="$(kubectl get nodes --no-headers | wc -l | tr -d ' ')"
SINGLE_NODE=false
if [[ "$NODE_COUNT" -lt 2 ]]; then
  log "WARN: single-node cluster; uncordon after evict for Job reschedule"
  SINGLE_NODE=true
fi

export PROMETHEUS_MOCK=true
export PROMETHEUS_MOCK_NODES="${FAULT_NODE}"
export HEALING_DRY_RUN=false
export POLL_INTERVAL=8s
export RESCHEDULE_TIMEOUT=90s
export TARGET_NAMESPACES=ai-training
export TRAINING_JOB_NAME=training-job
export TRAINING_JOB_NAMESPACE=ai-training
export METRICS_LISTEN="${METRICS_LISTEN:-:8080}"

log "starting operator locally (metrics ${METRICS_LISTEN})"
go run ./cmd/operator &
OP_PID=$!
trap 'kill ${OP_PID} 2>/dev/null || true' EXIT

log "waiting for node cordon"
for _ in $(seq 1 45); do
  if kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' 2>/dev/null | grep -q true; then
    break
  fi
  sleep 2
done
kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' | grep -q true || {
  echo "FAIL: node not cordoned" >&2
  exit 1
}

log "waiting for old pod termination"
for _ in $(seq 1 45); do
  kubectl get pod "$OLD_POD" -n ai-training >/dev/null 2>&1 || break
  sleep 2
done
kubectl get pod "$OLD_POD" -n ai-training >/dev/null 2>&1 && {
  echo "FAIL: old pod still exists" >&2
  exit 1
}

if [[ "$SINGLE_NODE" == "true" ]]; then
  log "uncordoning and removing GPU taint on ${FAULT_NODE} (single-node reschedule)"
  kubectl uncordon "$FAULT_NODE"
  kubectl taint nodes "$FAULT_NODE" ai-k8s-platform.io/gpu-fault:NoSchedule- 2>/dev/null || true
fi

NEW_POD=""
NEW_NODE=""
for _ in $(seq 1 60); do
  while read -r name node phase; do
    [[ "$phase" == "Running" && "$name" != "$OLD_POD" ]] || continue
    NEW_POD="$name"
    NEW_NODE="$node"
    break 2
  done < <(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
    -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName,PHASE:.status.phase --no-headers)
  sleep 2
done

if [[ -z "$NEW_POD" ]]; then
  echo "FAIL: no new Running pod" >&2
  kubectl get pods -n ai-training -o wide
  exit 1
fi

if [[ "$SINGLE_NODE" != "true" && "$NEW_NODE" == "$FAULT_NODE" ]]; then
  echo "FAIL: new pod still on fault node" >&2
  exit 1
fi

if kubectl get events -A 2>/dev/null | grep -qi Healing; then
  log "healing events present"
else
  log "WARN: no Healing events visible"
fi

kill "${OP_PID}" 2>/dev/null || true
trap - EXIT

log "P2 e2e-k3s PASSED (fault=${FAULT_NODE}, new=${NEW_POD}@${NEW_NODE}, single_node=${SINGLE_NODE})"
