#!/usr/bin/env bash
# Shared timing helpers for e2e / demo scripts.
set -euo pipefail

TIMING_EPOCH_START=""
declare -A TIMING_MARKS=()

timing_start() {
  TIMING_EPOCH_START="$(date +%s)"
  TIMING_MARKS=()
  TIMING_MARKS[start]="$TIMING_EPOCH_START"
}

timing_mark() {
  local name="$1"
  TIMING_MARKS["$name"]="$(date +%s)"
}

timing_delta() {
  local from="$1"
  local to="$2"
  local a="${TIMING_MARKS[$from]:-}"
  local b="${TIMING_MARKS[$to]:-}"
  if [[ -z "$a" || -z "$b" ]]; then
    echo "n/a"
    return
  fi
  echo "$((b - a))"
}

timing_print_summary() {
  local fault_node="${1:-}"
  local new_node="${2:-}"
  local new_pod="${3:-}"
  echo ""
  echo "=== Healing timing ==="
  if [[ -n "$fault_node" ]]; then
    echo "  fault_node:      ${fault_node}"
  fi
  if [[ -n "$new_pod" ]]; then
    echo "  new_pod:         ${new_pod}@${new_node:-unknown}"
  fi
  echo "  cordon_after_s:  $(timing_delta start cordon) (from trigger/start)"
  echo "  evict_done_s:    $(timing_delta start evicted) (from start, if marked)"
  echo "  reschedule_s:    $(timing_delta start recovered) (TTR from start)"
  echo "  cordon_to_recovered_s: $(timing_delta cordon recovered) (isolation → Running)"
  echo "  note: TTR includes PromQL poll interval + Job reschedule"
}
