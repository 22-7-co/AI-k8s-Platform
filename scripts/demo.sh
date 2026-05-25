#!/usr/bin/env bash
# P4 demo: one-command L1 walkthrough (k3s Plan B or kind L1-B).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

MODE="k3s"
DRY_RUN=false
METRICS_LISTEN="${METRICS_LISTEN:-:18081}"

log() { echo "==> $*"; }
warn() { echo "WARN: $*" >&2; }

usage() {
  cat <<'EOF'
AI-k8s-Platform — self-healing L1 demo

Usage:
  ./scripts/demo.sh              L1-A: current kubectl context (k3s / Plan B)
  ./scripts/demo.sh --kind       L1-B: kind 2-worker, strict node reschedule
  ./scripts/demo.sh --dry-run    Operator logs only (HEALING_DRY_RUN, no API writes)
  ./scripts/demo.sh --help

Environment:
  METRICS_LISTEN   Operator metrics bind (default :18081 for local demo)

After live demo, roll back:
  ./scripts/uncordon.sh <fault-node>

Docs: docs/demo-runbook.md  |  docs/interview-pitch.md

L2 (optional, not run by this script):
  Example Job + PVC can mount a checkpoint path for resume narrative;
  training framework owns resume logic — not an Operator feature.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    --kind) MODE="kind" ;;
    --dry-run) DRY_RUN=true ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

check_kubectl() {
  if ! command -v kubectl >/dev/null; then
    echo "ERROR: kubectl is required" >&2
    exit 1
  fi
  if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "ERROR: kubectl cannot reach a cluster (check context)" >&2
    exit 1
  fi
}

check_kind_deps() {
  if ! command -v docker >/dev/null || ! command -v kind >/dev/null; then
    echo "ERROR: --kind requires docker and kind" >&2
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    echo "ERROR: docker daemon is not running" >&2
    exit 1
  fi
}

print_snapshot() {
  echo ""
  echo "=== Cluster snapshot ==="
  kubectl get nodes -o wide 2>/dev/null || true
  kubectl get pods -n ai-training -o wide 2>/dev/null || true
  echo ""
  echo "Rollback:  ./scripts/uncordon.sh <fault-node>"
  echo "Pitch:     docs/interview-pitch.md"
}

run_dry_run() {
  check_kubectl
  log "dry-run mode (no cordon / evict / events)"
  kubectl apply -f "${ROOT}/deploy/manifests/operator/namespace.yaml"
  kubectl apply -f "${ROOT}/deploy/manifests/training/namespace.yaml"
  kubectl apply -f "${ROOT}/deploy/manifests/operator/serviceaccount.yaml"
  kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrole.yaml"
  kubectl apply -f "${ROOT}/deploy/manifests/operator/clusterrolebinding.yaml"
  kubectl apply -f "${ROOT}/deploy/manifests/training/job.yaml" 2>/dev/null || true
  kubectl wait --for=condition=Ready --timeout=120s -n ai-training \
    pod -l batch.kubernetes.io/job-name=training-job 2>/dev/null || true

  FAULT_NODE="$(kubectl get pods -n ai-training -l batch.kubernetes.io/job-name=training-job \
    -o jsonpath='{.items[0].spec.nodeName}' 2>/dev/null || kubectl get nodes -o jsonpath='{.items[0].metadata.name}')"
  if [[ -z "$FAULT_NODE" ]]; then
    echo "ERROR: no node to target for mock fault" >&2
    exit 1
  fi
  log "mock fault node: ${FAULT_NODE}"

  export PROMETHEUS_MOCK=true
  export PROMETHEUS_MOCK_NODES="${FAULT_NODE}"
  export HEALING_DRY_RUN=true
  export POLL_INTERVAL=5s
  export METRICS_LISTEN
  export TARGET_NAMESPACES=ai-training

  log "starting operator (15s sample)"
  go run ./cmd/operator 2>&1 &
  OP_PID=$!
  trap 'kill ${OP_PID} 2>/dev/null || true' EXIT
  sleep 15
  kill "${OP_PID}" 2>/dev/null || true
  trap - EXIT

  if kubectl get node "$FAULT_NODE" -o jsonpath='{.spec.unschedulable}' 2>/dev/null | grep -q true; then
    echo "FAIL: dry-run cordoned the node" >&2
    exit 1
  fi
  log "dry-run OK — node ${FAULT_NODE} still schedulable"
  print_snapshot
}

run_k3s() {
  check_kubectl
  log "L1-A demo (delegates to e2e-k3s.sh, Plan B on single-node)"
  log "metrics on ${METRICS_LISTEN}"
  export METRICS_LISTEN
  "${ROOT}/scripts/e2e-k3s.sh"
  print_snapshot
  log "demo finished — see e2e-k3s output above"
}

run_kind() {
  check_kind_deps
  log "L1-B demo (delegates to e2e-kind.sh)"
  "${ROOT}/scripts/e2e-kind.sh"
  log "demo finished — kind cluster removed unless KEEP_CLUSTER=true"
}

echo "AI-k8s-Platform demo (mode=${MODE}, dry_run=${DRY_RUN})"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
  run_dry_run
elif [[ "$MODE" == "kind" ]]; then
  run_kind
else
  run_k3s
fi
