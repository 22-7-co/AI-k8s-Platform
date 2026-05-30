#!/usr/bin/env bash
# L3 cloud interview demo: deploy stack (optional), run e2e-cloud, print recording checklist.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { echo "==> $*"; }

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  cat <<'EOF'
AI-k8s-Platform — L3 cloud demo (3 VM k3s)

Usage:
  export KUBECONFIG=~/.kube/config-cloud
  ./scripts/demo-cloud.sh

Env:
  DEPLOY_STACK=false   skip deploy if stack already up

Docs: docs/cloud-lab.md
EOF
  exit 0
fi

if ! kubectl cluster-info >/dev/null 2>&1; then
  echo "ERROR: set KUBECONFIG to cloud k3s cluster (see docs/cloud-lab.md)" >&2
  exit 1
fi

log "cloud cluster nodes"
kubectl get nodes -o wide

export DEPLOY_STACK="${DEPLOY_STACK:-true}"
"${ROOT}/scripts/e2e-cloud.sh"

echo ""
echo "=== Cluster snapshot ==="
kubectl get pods -n ai-training -o wide
echo ""
echo "=== Recording checklist ==="
echo "  Grafana: port-forward prometheus / control :3000 — Dashboard AI 平台自愈监控"
echo "  kubectl get pods -o wide -n ai-training"
echo "  Rollback: ./scripts/uncordon.sh <fault-node>"
echo "  Pitch: docs/interview-pitch.md"
