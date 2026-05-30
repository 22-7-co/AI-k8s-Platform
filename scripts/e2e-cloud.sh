#!/usr/bin/env bash
# L3 cloud e2e: real PromQL + Exporter inject + strict node reschedule (2+ workers).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=lib/timing.sh
source "${ROOT}/scripts/lib/timing.sh"

DEPLOY_STACK="${DEPLOY_STACK:-true}"
MIN_WORKERS="${MIN_WORKERS:-2}"

log() { echo "==> $*"; }

inject_xid() {
  local node="$1"
  local ip
  ip="$(kubectl get node "$node" -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}')"
  if [[ -z "$ip" ]]; then
    echo "FAIL: no InternalIP for node ${node}" >&2
    return 1
  fi
  log "inject XID on ${node} via ${ip}:9100"
  curl -sf -m 10 -X POST "http://${ip}:9100/inject/xid?node=${node}&gpu_id=0&xid_code=79"
}

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "ERROR: kubectl cannot reach cluster (cloud k3s — see docs/cloud-lab.md)" >&2
  exit 1
fi

WORKERS="$(kubectl get nodes -o jsonpath='{range .items[?(@.metadata.labels.node-role\.kubernetes\.io/control-plane!="true")]}{.metadata.name}{"\n"}{end}' | grep -c . || true)"
if [[ "${WORKERS:-0}" -lt "$MIN_WORKERS" ]]; then
  echo "ERROR: need at least ${MIN_WORKERS} worker nodes, found ${WORKERS:-0}" >&2
  kubectl get nodes -o wide
  exit 1
fi

if [[ "$DEPLOY_STACK" == "true" ]]; then
  "${ROOT}/scripts/cloud/deploy-stack.sh"
fi

log "reset worker healing state"
while read -r n; do
  [[ -n "$n" ]] || continue
  kubectl uncordon "$n" 2>/dev/null || true
  kubectl taint nodes "$n" ai-k8s-platform.io/gpu-fault:NoSchedule- 2>/dev/null || true
  kubectl label node "$n" ai-k8s-platform.io/healing-state- 2>/dev/null || true
  kubectl annotate node "$n" ai-k8s-platform.io/healing-completed-at- 2>/dev/null || true
done < <(kubectl get nodes -o jsonpath='{range .items[?(@.metadata.labels.node-role\.kubernetes\.io/control-plane=="")]}{.metadata.name}{"\n"}{end}')

kubectl delete job training-job -n ai-training --ignore-not-found --wait=true 2>/dev/null || true
sleep 2
kubectl apply -f "${ROOT}/deploy/manifests/training/job.yaml"
kubectl wait --for=condition=Ready --timeout=180s -n ai-training \
  pod -l batch.kubernetes.io/job-name=training-job

FAULT_NODE="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
  -o jsonpath='{.items[0].spec.nodeName}')"
OLD_POD="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
  -o jsonpath='{.items[0].metadata.name}')"
log "training pod ${OLD_POD} on ${FAULT_NODE}"

log "ensure operator uses real PromQL"
kubectl apply -f "${ROOT}/deploy/manifests/operator/configmap-cloud.yaml"
kubectl rollout restart deployment/ai-operator -n ai-platform
kubectl rollout status deployment/ai-operator -n ai-platform --timeout=180s

timing_start
inject_xid "$FAULT_NODE"

log "waiting for node cordon"
for _ in $(seq 1 90); do
  if kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' 2>/dev/null | grep -q true; then
    break
  fi
  sleep 2
done
kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' | grep -q true || {
  echo "FAIL: node not cordoned after PromQL inject" >&2
  kubectl logs -n ai-platform deployment/ai-operator --tail=80 || true
  exit 1
}
timing_mark cordon

for _ in $(seq 1 90); do
  kubectl get pod "$OLD_POD" -n ai-training >/dev/null 2>&1 || break
  sleep 2
done
kubectl get pod "$OLD_POD" -n ai-training >/dev/null 2>&1 && {
  echo "FAIL: old pod still exists" >&2
  exit 1
}
timing_mark evicted

NEW_NODE=""
NEW_POD=""
for _ in $(seq 1 120); do
  NEW_POD="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
    -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' 2>/dev/null | awk '{print $1}')"
  if [[ -n "$NEW_POD" && "$NEW_POD" != "$OLD_POD" ]]; then
    NEW_NODE="$(kubectl get pod "$NEW_POD" -n ai-training -o jsonpath='{.spec.nodeName}')"
    if [[ -n "$NEW_NODE" && "$NEW_NODE" != "$FAULT_NODE" ]]; then
      break
    fi
  fi
  sleep 2
done

if [[ -z "$NEW_NODE" || "$NEW_NODE" == "$FAULT_NODE" ]]; then
  echo "FAIL: no Running pod on a healthy worker" >&2
  kubectl get pods -n ai-training -o wide
  exit 1
fi

timing_mark recovered
timing_print_summary "$FAULT_NODE" "$NEW_NODE" "$NEW_POD"
log "L3 e2e-cloud PASSED (fault=${FAULT_NODE}, new=${NEW_POD}@${NEW_NODE})"
