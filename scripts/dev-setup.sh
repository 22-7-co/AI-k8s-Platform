#!/usr/bin/env bash
# Local dev prerequisites check (optional tools).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> AI-k8s-Platform dev setup"
command -v go >/dev/null && go version || echo "WARN: go not found"
command -v kubectl >/dev/null && kubectl version --client=true 2>/dev/null || echo "WARN: kubectl not found"

if [[ ! -f .env && -f .env.example ]]; then
  echo "Tip: cp .env.example .env for local config"
fi

echo "==> done"
