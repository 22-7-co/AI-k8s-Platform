#!/usr/bin/env bash
# P3 L1-B e2e: kind 2-worker cluster, in-cluster operator Deployment, strict node reschedule.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CLUSTER="${KIND_CLUSTER_NAME:-ai-k8s-e2e}"
OPERATOR_IMAGE="${OPERATOR_IMAGE:-ai-k8s-platform/operator:dev}"
KEEP_CLUSTER="${KEEP_CLUSTER:-false}"
RUN_PROMQL_E2E="${RUN_PROMQL_E2E:-false}"

log() { echo "==> $*"; }

if ! command -v kind >/dev/null || ! command -v kubectl >/dev/null || ! command -v docker >/dev/null; then
  echo "ERROR: kind, kubectl, and docker are required" >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "ERROR: docker daemon is not running" >&2
  exit 1
fi

if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  log "creating kind cluster $CLUSTER (2 workers)"
  kind create cluster --name "$CLUSTER" --config "${ROOT}/scripts/kind-e2e-config.yaml"
fi

kubectl config use-context "kind-${CLUSTER}"

TRAINING_IMAGE="${TRAINING_IMAGE:-busybox:1.36}"

load_image_to_kind() {
  local img="$1"
  log "preload image ${img} into kind nodes"
  docker pull "$img" >/dev/null 2>&1 || true
  if kind load docker-image "$img" --name "$CLUSTER" 2>/dev/null; then
    return 0
  fi
  while read -r n; do
    [[ -n "$n" ]] || continue
    docker save "$img" | docker exec -i "$n" ctr --namespace=k8s.io images import - >/dev/null 2>&1 || true
  done < <(kind get nodes --name "$CLUSTER")
}

log "building operator image"
docker build -f "${ROOT}/deploy/docker/Dockerfile.operator" -t "$OPERATOR_IMAGE" "$ROOT"
load_image_to_kind "$OPERATOR_IMAGE"
load_image_to_kind "$TRAINING_IMAGE"

log "applying manifests"
kubectl apply -f "${ROOT}/deploy/manifests/operator/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/training/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/serviceaccount.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrole.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrolebinding.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/configmap.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/deployment.yaml"

log "reset worker nodes (repeatable e2e)"
while read -r n; do
  [[ -n "$n" ]] || continue
  kubectl uncordon "$n" 2>/dev/null || true
  kubectl taint nodes "$n" ai-k8s-platform.io/gpu-fault:NoSchedule- 2>/dev/null || true
  kubectl label node "$n" ai-k8s-platform.io/healing-state- 2>/dev/null || true
  kubectl annotate node "$n" ai-k8s-platform.io/healing-completed-at- 2>/dev/null || true
done < <(kubectl get nodes -o jsonpath='{range .items[?(@.metadata.labels.node-role\.kubernetes\.io/control-plane=="")]}{.metadata.name}{"\n"}{end}')

log "cleaning previous training job"
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
log "training pod ${OLD_POD} on node ${FAULT_NODE}"

log "configuring in-cluster operator mock fault on ${FAULT_NODE}"
kubectl patch configmap operator-config -n ai-platform --type merge \
  -p "{\"data\":{\"PROMETHEUS_MOCK\":\"true\",\"PROMETHEUS_MOCK_NODES\":\"${FAULT_NODE}\",\"HEALING_DRY_RUN\":\"false\",\"POLL_INTERVAL\":\"8s\",\"RESCHEDULE_TIMEOUT\":\"120s\"}}"
kubectl set image deployment/ai-operator -n ai-platform operator="$OPERATOR_IMAGE" --record=false 2>/dev/null || true
kubectl rollout restart deployment/ai-operator -n ai-platform
kubectl rollout status deployment/ai-operator -n ai-platform --timeout=120s
kubectl wait --for=condition=Ready --timeout=120s -n ai-platform pod -l app.kubernetes.io/name=ai-operator

if ! kubectl logs -n ai-platform deployment/ai-operator --tail=30 2>/dev/null | grep -q action_id; then
  echo "WARN: operator logs missing action_id in last 30 lines (checking after heal)"
fi

log "waiting for node cordon"
for _ in $(seq 1 90); do
  if kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' | grep -q true; then
    break
  fi
  sleep 2
done
if ! kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' | grep -q true; then
  echo "FAIL: node ${FAULT_NODE} not cordoned" >&2
  kubectl logs -n ai-platform deployment/ai-operator --tail=80 || true
  exit 1
fi

log "waiting for old pod termination"
for _ in $(seq 1 90); do
  if kubectl get pod "$OLD_POD" -n ai-training >/dev/null 2>&1; then
    sleep 2
  else
    break
  fi
done
if kubectl get pod "$OLD_POD" -n ai-training >/dev/null 2>&1; then
  echo "FAIL: old pod still exists" >&2
  exit 1
fi

log "waiting for new pod on another node"
NEW_NODE=""
for _ in $(seq 1 120); do
  NEW_POD="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
    -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' 2>/dev/null | awk '{print $1}')"
  if [[ -n "${NEW_POD}" ]]; then
    NEW_NODE="$(kubectl get pod "$NEW_POD" -n ai-training -o jsonpath='{.spec.nodeName}')"
    if [[ -n "$NEW_NODE" && "$NEW_NODE" != "$FAULT_NODE" ]]; then
      log "new pod ${NEW_POD} running on ${NEW_NODE}"
      break
    fi
  fi
  sleep 2
done

if [[ -z "$NEW_NODE" || "$NEW_NODE" == "$FAULT_NODE" ]]; then
  echo "FAIL: no Running pod on a healthy node" >&2
  kubectl get pods -n ai-training -o wide
  kubectl logs -n ai-platform deployment/ai-operator --tail=100 || true
  exit 1
fi

if ! kubectl logs -n ai-platform deployment/ai-operator --tail=50 | grep -q action_id; then
  echo "FAIL: operator logs missing action_id" >&2
  kubectl logs -n ai-platform deployment/ai-operator --tail=50 || true
  exit 1
fi

EVENTS="$(kubectl get events -A --field-selector involvedObject.name="${FAULT_NODE}" 2>/dev/null | grep -i Healing || true)"
if [[ -z "$EVENTS" ]]; then
  EVENTS="$(kubectl get events -A | grep -i Healing | tail -5 || true)"
fi
if [[ -z "$EVENTS" ]]; then
  echo "WARN: no Healing events found (non-fatal)"
else
  log "healing events present"
fi

log "P3 e2e-kind PASSED (fault=${FAULT_NODE}, new=${NEW_NODE})"

if [[ "$RUN_PROMQL_E2E" == "true" ]]; then
  log "running real PromQL sub-gate (KEEP_CLUSTER implied)"
  export KEEP_CLUSTER=true
  "${ROOT}/scripts/e2e-promql.sh"
fi

if [[ "$KEEP_CLUSTER" != "true" ]]; then
  log "deleting kind cluster ${CLUSTER}"
  kind delete cluster --name "$CLUSTER"
fi
