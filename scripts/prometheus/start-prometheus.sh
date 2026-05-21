#!/usr/bin/env bash
# Start/stop a local Prometheus container that scrapes the mock exporter on :9100.
#
# Prerequisites: docker or podman. Exporter must listen on host :9100 (./bin/exporter).
# Linux: maps host.docker.internal via --add-host=host.docker.internal:host-gateway.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIG="${ROOT}/scripts/prometheus/prometheus.yml"
CONTAINER_NAME="${PROMETHEUS_CONTAINER:-ai-k8s-prometheus}"
IMAGE="${PROMETHEUS_IMAGE:-prom/prometheus:v2.51.2}"
PORT="${PROMETHEUS_PORT:-9090}"

runtime() {
  if command -v docker >/dev/null 2>&1; then
    echo docker
    return
  fi
  if command -v podman >/dev/null 2>&1; then
    echo podman
    return
  fi
  echo ""
}

cmd_up() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    echo "ERROR: docker or podman required. Install one of them or run Prometheus manually with ${CONFIG}" >&2
    exit 1
  fi
  if $rt ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    $rt rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  $rt run -d --name "$CONTAINER_NAME" \
    -p "${PORT}:9090" \
    --add-host=host.docker.internal:host-gateway \
    -v "${CONFIG}:/etc/prometheus/prometheus.yml:ro" \
    "$IMAGE" \
    --config.file=/etc/prometheus/prometheus.yml \
    --web.enable-lifecycle
  echo "Prometheus UI: http://localhost:${PORT}"
  echo "Targets:       http://localhost:${PORT}/targets"
}

cmd_down() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    exit 0
  fi
  $rt rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  echo "stopped ${CONTAINER_NAME}"
}

cmd_check() {
  curl -sf "http://localhost:${PORT}/-/healthy" && echo "prometheus healthy"
  curl -sf "http://localhost:${PORT}/api/v1/targets" | head -c 200
  echo ""
}

usage() {
  echo "usage: $0 {up|down|check}"
}

case "${1:-}" in
  up) cmd_up ;;
  down) cmd_down ;;
  check) cmd_check ;;
  *) usage; exit 1 ;;
esac
