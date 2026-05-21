# deploy/manifests

Place Kubernetes YAML here:

- `dcgm-daemonset.yaml` — per-node DCGM
- `exporter-daemonset.yaml` — Go metrics exporter
- `operator/` — Deployment, ServiceAccount, ClusterRole(Binding)

Keep RBAC aligned with verbs used in `internal/healing` and `internal/controller`.
