# deploy/manifests

Kubernetes YAML for operator, exporter, observability, and training workloads.

| Path | Purpose |
|------|---------|
| `operator/` | Operator Deployment, RBAC, ConfigMap |
| `operator/configmap-cloud.yaml` | L3 cloud: real PromQL, in-cluster Prometheus URL |
| `exporter/daemonset.yaml` | GPU metrics Exporter (hostNetwork :9100) |
| `observability/prometheus.yaml` | In-cluster Prometheus for cloud lab |
| `training/job.yaml` | L1 training Job |
| `training/pvc.yaml` | L2 checkpoint PVC |
| `training/job-with-checkpoint.yaml` | L2 Job mounting PVC |

Keep RBAC aligned with verbs used in `internal/healing` and `internal/controller`.

Cloud deploy: `./scripts/cloud/deploy-stack.sh`
