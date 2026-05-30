#!/usr/bin/env bash
# Deploy operator, exporter DS, prometheus, training job on existing k3s cluster (L3 cloud).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

OPERATOR_IMAGE="${OPERATOR_IMAGE:-ai-k8s-platform/operator:dev}"
EXPORTER_IMAGE="${EXPORTER_IMAGE:-ai-k8s-platform/exporter:dev}"
TRAINING_IMAGE="${TRAINING_IMAGE:-busybox:1.36}"
BUILD_IMAGES="${BUILD_IMAGES:-true}"
LOAD_IMAGES="${LOAD_IMAGES:-false}"

log() { echo "==> $*"; }

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "ERROR: kubectl cannot reach cluster (set KUBECONFIG)" >&2
  exit 1
fi

if [[ "$BUILD_IMAGES" == "true" ]]; then
  log "building images"
  docker build -f "${ROOT}/deploy/docker/Dockerfile.operator" -t "$OPERATOR_IMAGE" "$ROOT"
  docker build -f "${ROOT}/deploy/docker/Dockerfile.exporter" -t "$EXPORTER_IMAGE" "$ROOT"
fi

if [[ "$LOAD_IMAGES" == "true" ]]; then
  log "LOAD_IMAGES=true: import images into k3s containerd (single-node dev)"
  docker save "$OPERATOR_IMAGE" | sudo k3s ctr images import - 2>/dev/null || true
  docker save "$EXPORTER_IMAGE" | sudo k3s ctr images import - 2>/dev/null || true
  docker pull "$TRAINING_IMAGE" >/dev/null 2>&1 || true
  docker save "$TRAINING_IMAGE" | sudo k3s ctr images import - 2>/dev/null || true
fi

log "applying manifests"
kubectl apply -f "${ROOT}/deploy/manifests/operator/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/training/namespace.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/serviceaccount.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrole.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrolebinding.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/operator/configmap-cloud.yaml"
kubectl apply -f "${ROOT}/deploy/manifests/observability/prometheus.yaml"

sed "s|ai-k8s-platform/operator:dev|${OPERATOR_IMAGE}|g" "${ROOT}/deploy/manifests/operator/deployment.yaml" | kubectl apply -f -
sed "s|ai-k8s-platform/exporter:dev|${EXPORTER_IMAGE}|g" "${ROOT}/deploy/manifests/exporter/daemonset.yaml" | kubectl apply -f -

kubectl apply -f "${ROOT}/deploy/manifests/training/job.yaml"

log "waiting for core pods"
kubectl rollout status deployment/ai-operator -n ai-platform --timeout=180s
kubectl rollout status deployment/prometheus -n ai-platform --timeout=180s
kubectl rollout status daemonset/gpu-metrics-exporter -n ai-platform --timeout=180s || true

kubectl wait --for=condition=Ready --timeout=180s -n ai-training \
  pod -l batch.kubernetes.io/job-name=training-job 2>/dev/null || true

log "stack deployed"
kubectl get pods -A -o wide | grep -E 'ai-platform|ai-training' || kubectl get pods -A
