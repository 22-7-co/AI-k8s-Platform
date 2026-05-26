#!/usr/bin/env bash
# Start/stop a local Prometheus container scraping host Exporter (:9100) and Operator (:18081).
#
# Prerequisites: docker or podman. Host must run ./bin/exporter and operator (METRICS_LISTEN=:18081).
# Linux: maps host.docker.internal via --add-host=host.docker.internal:host-gateway.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATE="${ROOT}/scripts/prometheus/prometheus.yml"
RUNTIME_CONFIG="${ROOT}/scripts/prometheus/prometheus.runtime.yml"
CONTAINER_NAME="${PROMETHEUS_CONTAINER:-ai-k8s-prometheus}"
IMAGE="${PROMETHEUS_IMAGE:-prom/prometheus:v2.51.2}"
PORT="${PROMETHEUS_PORT:-9090}"
DOCKER_NETWORK="${DOCKER_NETWORK:-}"
EXPORTER_METRICS_HOST="${EXPORTER_METRICS_HOST:-host.docker.internal:9100}"
OPERATOR_METRICS_HOST="${OPERATOR_METRICS_HOST:-host.docker.internal:18081}"

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

render_config() {
  export EXPORTER_METRICS_HOST OPERATOR_METRICS_HOST
  if ! command -v envsubst >/dev/null 2>&1; then
    echo "ERROR: envsubst required (gettext package)" >&2
    exit 1
  fi
  envsubst '${EXPORTER_METRICS_HOST} ${OPERATOR_METRICS_HOST}' < "$TEMPLATE" > "$RUNTIME_CONFIG"
}

cmd_up() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    echo "ERROR: docker or podman required. Install one of them or run Prometheus manually." >&2
    exit 1
  fi
  render_config
  if $rt ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    $rt rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  local -a run_args=(
    run -d --name "$CONTAINER_NAME"
    -p "${PORT}:9090"
    --add-host=host.docker.internal:host-gateway
    -v "${RUNTIME_CONFIG}:/etc/prometheus/prometheus.yml:ro"
  )
  if [[ -n "$DOCKER_NETWORK" ]]; then
    run_args+=(--network "$DOCKER_NETWORK")
  fi
  $rt "${run_args[@]}" "$IMAGE" \
    --config.file=/etc/prometheus/prometheus.yml \
    --web.enable-lifecycle
  echo "Prometheus UI: http://localhost:${PORT}"
  echo "Targets:       http://localhost:${PORT}/targets"
  echo "Scrape:        exporter=${EXPORTER_METRICS_HOST} operator=${OPERATOR_METRICS_HOST}"
}

cmd_down() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    exit 0
  fi
  $rt rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  rm -f "$RUNTIME_CONFIG" 2>/dev/null || true
  echo "stopped ${CONTAINER_NAME}"
}

cmd_check() {
  curl -sf "http://localhost:${PORT}/-/healthy" && echo "prometheus healthy"
  if command -v jq >/dev/null 2>&1; then
    curl -sf "http://localhost:${PORT}/api/v1/targets" | \
      jq '.data.activeTargets[] | {job: .labels.job, health: .health}'
  else
    curl -sf "http://localhost:${PORT}/api/v1/targets" | head -c 400
    echo ""
  fi
}

usage() {
  echo "usage: $0 {up|down|check}"
  echo "  OPERATOR_METRICS_HOST  default host.docker.internal:18081"
  echo "  EXPORTER_METRICS_HOST  default host.docker.internal:9100"
}

case "${1:-}" in
  up) cmd_up ;;
  down) cmd_down ;;
  check) cmd_check ;;
  *) usage; exit 1 ;;
esac
