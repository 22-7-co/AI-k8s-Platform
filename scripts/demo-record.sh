#!/usr/bin/env bash
# P4 interview recording flow: observability stack + demo + Grafana hints.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { echo "==> $*"; }

MODE="${1:-kind}"
case "$MODE" in
  --help|-h)
    cat <<'EOF'
Usage:
  ./scripts/demo-record.sh           # kind L1-B + observability hints
  ./scripts/demo-record.sh k3s       # k3s L1-A (Plan B)
  ./scripts/demo-record.sh cloud     # print cloud recording checklist only

Opens Grafana http://localhost:3000 — Dashboard "AI 平台自愈监控"
Set time range: Last 15m, refresh 10s. Record XID spike + healing actions + verify.
EOF
    exit 0
    ;;
  k3s) DEMO_ARGS=() ;;
  kind) DEMO_ARGS=(--kind) ;;
  cloud)
    log "Cloud recording checklist (cluster must already exist — see docs/cloud-lab.md)"
    echo "  1. kubectl get nodes -o wide"
    echo "  2. ./scripts/demo-cloud.sh"
    echo "  3. Grafana on control: http://<control-ip>:3000"
    echo "  4. kubectl get pods -o wide -n ai-training"
    exit 0
    ;;
  *)
    echo "unknown mode: $MODE (use kind, k3s, or cloud)" >&2
    exit 1
    ;;
esac

if [[ "$MODE" != "cloud" ]]; then
  if ! command -v docker >/dev/null; then
    echo "WARN: docker not found; skip observability stack" >&2
  else
    log "starting observability stack (Prometheus + Grafana)"
    "${ROOT}/scripts/observability-stack.sh" up || true
    echo ""
    echo "Grafana: http://localhost:3000  (admin/admin)"
    echo "Prometheus targets: http://localhost:9090/targets"
    echo "Dashboard: AI 平台自愈监控 — Last 15m, refresh 10s"
    echo ""
  fi

  log "running demo ${DEMO_ARGS[*]:-}"
  "${ROOT}/scripts/demo.sh" "${DEMO_ARGS[@]}"

  echo ""
  log "recording tips"
  echo "  - Re-run inject/demo if action rate graph is flat; capture the spike window"
  echo "  - Show: operator_up=1, XID panel, healing actions, seconds since last success"
  echo "  - Terminal: kubectl get pods -o wide -n ai-training"
  echo "  - Rollback: ./scripts/uncordon.sh <fault-node>"
fi
