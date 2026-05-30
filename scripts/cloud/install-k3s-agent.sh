#!/usr/bin/env bash
# Join a VM to an existing k3s server (run on worker as root).
set -euo pipefail

usage() {
  cat <<'EOF'
Usage (on worker VM):
  K3S_URL=https://<control-private-ip>:6443 K3S_TOKEN=<token> ./install-k3s-agent.sh

Get token on control:
  sudo cat /var/lib/rancher/k3s/server/node-token
EOF
}

if [[ "${EUID:-0}" -ne 0 ]]; then
  echo "run as root on worker" >&2
  usage >&2
  exit 1
fi

: "${K3S_URL:?set K3S_URL=https://control:6443}"
: "${K3S_TOKEN:?set K3S_TOKEN from control node-token}"

curl -sfL https://get.k3s.io | sh -
echo "worker joined; verify on control: kubectl get nodes"
