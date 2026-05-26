#!/usr/bin/env bash
# Start/stop local Grafana with provisioning for AI Platform Healing dashboard.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GRAFANA_PROVISIONING="${ROOT}/docs/examples/grafana/provisioning"
GRAFANA_DASHBOARDS="${ROOT}/docs/examples/grafana/dashboards"
DATASOURCE_TEMPLATE="${GRAFANA_PROVISIONING}/datasources/prometheus.yml"
DATASOURCE_RUNTIME="${GRAFANA_PROVISIONING}/datasources/prometheus.runtime.yml"
CONTAINER_NAME="${GRAFANA_CONTAINER:-ai-k8s-grafana}"
IMAGE="${GRAFANA_IMAGE:-grafana/grafana:10.4.2}"
PORT="${GRAFANA_PORT:-3000}"
DOCKER_NETWORK="${DOCKER_NETWORK:-}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://host.docker.internal:9090}"

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

render_datasource() {
  export PROMETHEUS_URL
  if ! command -v envsubst >/dev/null 2>&1; then
    echo "ERROR: envsubst required (gettext package)" >&2
    exit 1
  fi
  envsubst '${PROMETHEUS_URL}' < "$DATASOURCE_TEMPLATE" > "$DATASOURCE_RUNTIME"
}

cmd_up() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    echo "ERROR: docker or podman required" >&2
    exit 1
  fi
  render_datasource
  if $rt ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    $rt rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  local -a run_args=(
    run -d --name "$CONTAINER_NAME"
    -p "${PORT}:3000"
    --add-host=host.docker.internal:host-gateway
    -v "${DATASOURCE_RUNTIME}:/etc/grafana/provisioning/datasources/prometheus.yml:ro"
    -v "${GRAFANA_PROVISIONING}/dashboards:/etc/grafana/provisioning/dashboards:ro"
    -v "${GRAFANA_DASHBOARDS}:/var/lib/grafana/dashboards:ro"
    -e GF_SECURITY_ADMIN_USER=admin
    -e GF_SECURITY_ADMIN_PASSWORD=admin
    -e GF_USERS_ALLOW_SIGN_UP=false
  )
  if [[ -n "$DOCKER_NETWORK" ]]; then
    run_args+=(--network "$DOCKER_NETWORK")
  fi
  $rt "${run_args[@]}" "$IMAGE"
  echo "Grafana UI: http://localhost:${PORT}  (admin / admin, local demo only)"
  echo "Dashboard:  AI Platform Healing"
  echo "Prometheus: ${PROMETHEUS_URL}"
}

cmd_down() {
  local rt
  rt="$(runtime)"
  if [[ -z "$rt" ]]; then
    exit 0
  fi
  $rt rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  rm -f "$DATASOURCE_RUNTIME" 2>/dev/null || true
  echo "stopped ${CONTAINER_NAME}"
}

cmd_check() {
  curl -sf "http://localhost:${PORT}/api/health" | head -c 200
  echo ""
  echo "grafana healthy"
}

usage() {
  echo "usage: $0 {up|down|check}"
  echo "  PROMETHEUS_URL  default http://host.docker.internal:9090"
}

case "${1:-}" in
  up) cmd_up ;;
  down) cmd_down ;;
  check) cmd_check ;;
  *) usage; exit 1 ;;
esac
