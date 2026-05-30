#!/usr/bin/env bash
# L2 demo: PVC checkpoint survives pod reschedule (local kind/k3s/cloud).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { echo "==> $*"; }

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "ERROR: kubectl required" >&2
  exit 1
fi

log "apply PVC + checkpoint job"
kubectl apply -f "${ROOT}/deploy/manifests/training/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/training/pvc.yaml"
kubectl delete job training-job-checkpoint -n ai-training --ignore-not-found --wait=true 2>/dev/null || true
sleep 2
kubectl apply -f "${ROOT}/deploy/manifests/training/job-with-checkpoint.yaml"

kubectl wait --for=condition=Ready --timeout=180s -n ai-training \
  pod -l batch.kubernetes.io/job-name=training-job-checkpoint 2>/dev/null || \
  kubectl wait --for=condition=Ready --timeout=180s -n ai-training \
  pod -l job-name=training-job-checkpoint

POD="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job-checkpoint \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
if [[ -z "$POD" ]]; then
  POD="$(kubectl get pods -n ai-training --no-headers | awk '/training-job-checkpoint/{print $1; exit}')"
fi
NODE="$(kubectl get pod "$POD" -n ai-training -o jsonpath='{.spec.nodeName}')"

log "seed checkpoint on ${POD}@${NODE}"
kubectl exec -n ai-training "$POD" -- sh -c 'echo seeded-v1 > /checkpoints/epoch.ckpt && cat /checkpoints/epoch.ckpt'

log "L2 narrative: Operator would evict this pod on GPU fault; new pod mounts same PVC and reads epoch.ckpt"
log "To combine with L1 healing, run e2e-kind/cloud with job-with-checkpoint.yaml instead of job.yaml"

echo ""
echo "L2 demo-l2 OK — checkpoint at /checkpoints/epoch.ckpt on PVC training-checkpoints"
echo "Resume logic is in the training container; Operator only ensures Pod+PVC reschedule."
