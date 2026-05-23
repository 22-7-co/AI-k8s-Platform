#!/usr/bin/env bash
# Roll back node isolation after a false positive or approved recovery.
# Usage: ./scripts/uncordon.sh <node-name>
set -euo pipefail

NODE="${1:-}"
if [[ -z "$NODE" ]]; then
  echo "usage: $0 <node-name>" >&2
  exit 1
fi

if ! kubectl get node "$NODE" >/dev/null 2>&1; then
  echo "ERROR: node ${NODE} not found" >&2
  exit 1
fi

echo "==> uncordon ${NODE}"
kubectl uncordon "$NODE"

echo "==> remove GPU fault taint"
kubectl taint nodes "$NODE" ai-k8s-platform.io/gpu-fault:NoSchedule- 2>/dev/null || true

echo "==> clear healing labels/annotations"
kubectl label node "$NODE" ai-k8s-platform.io/healing-state- 2>/dev/null || true
kubectl annotate node "$NODE" \
  ai-k8s-platform.io/healing-completed-at- \
  ai-k8s-platform.io/healing-fail-count- 2>/dev/null || true

kubectl get node "$NODE" -o custom-columns=NAME:.metadata.name,UNSCHED:.spec.unschedulable,STATE:.metadata.labels.ai-k8s-platform\.io/healing-state

echo "==> done"
