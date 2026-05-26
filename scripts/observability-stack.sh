#!/usr/bin/env bash
# One-shot local Prometheus + Grafana for demo (read-only observability).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NETWORK="${OBSERVABILITY_NETWORK:-ai-k8s-observability}"

runtime() {
  if command -v docker >/dev/null 2>&1; then echo docker; return; fi
  if command -v podman >/dev/null 2>&1; then echo podman; return; fi
  echo ""
}

ensure_network() {
  local rt="$1"
  if [[ -z "$rt" ]]; then
    return 0
  fi
  if ! $rt network inspect "$NETWORK" >/dev/null 2>&1; then
    $rt network create "$NETWORK" >/dev/null
    echo "created docker network ${NETWORK}"
  fi
}

cmd_up() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    echo "ERROR: docker or podman required" >&2
    exit 1
  fi
  ensure_network "$rt"
  export DOCKER_NETWORK="$NETWORK"
  export PROMETHEUS_URL="http://ai-k8s-prometheus:9090"
  "${ROOT}/scripts/prometheus/start-prometheus.sh" up
  "${ROOT}/scripts/grafana/start-grafana.sh" up
  echo ""
  echo "==> Observability stack is up"
  echo "    Prometheus: http://localhost:9090"
  echo "    Grafana:    http://localhost:3000  (admin/admin)"
  echo ""
  echo "Host prerequisites (separate terminals):"
  echo "  ./bin/exporter &"
  echo "  METRICS_LISTEN=:18081 go run ./cmd/operator &"
}

cmd_down() {
  "${ROOT}/scripts/grafana/start-grafana.sh" down || true
  "${ROOT}/scripts/prometheus/start-prometheus.sh" down || true
  local rt
  rt="$(runtime)"
  if [[ -n "$rt" ]]; then
    $rt network rm "$NETWORK" >/dev/null 2>&1 || true
  fi
  echo "observability stack stopped"
}

cmd_status() {
  echo "==> Prometheus"
  "${ROOT}/scripts/prometheus/start-prometheus.sh" check || echo "Prometheus not healthy"
  echo ""
  echo "==> Grafana"
  "${ROOT}/scripts/grafana/start-grafana.sh" check || echo "Grafana not healthy"
}

usage() {
  echo "usage: $0 {up|down|status}"
}

case "${1:-}" in
  up) cmd_up ;;
  down) cmd_down ;;
  status) cmd_status ;;
  *) usage; exit 1 ;;
esac
