# AI-k8s-Platform — Agent Guide

## Project

Self-healing AI compute platform on Kubernetes: detect GPU hardware faults (DCGM / Prometheus), cordon/taint nodes, evict unhealthy training pods, and rely on Job + checkpoint storage for minute-level recovery.

See [说明文档.txt](./说明文档.txt) and [docs/architecture.md](./docs/architecture.md) for product context.

## Repository layout

| Path | Purpose |
|------|---------|
| `cmd/operator/` | Kubernetes controller / operator entrypoint |
| `cmd/exporter/` | DCGM → Prometheus metrics exporter (Go) |
| `internal/controller/` | Reconcilers, watches (Pods, Nodes) |
| `internal/healing/` | Cordon, taint, eviction orchestration |
| `internal/prometheus/` | Alert / metric query client |
| `api/` | CRD types (if using kubebuilder) |
| `pkg/` | Shared libraries safe for external import |
| `deploy/manifests/` | Raw K8s YAML (DaemonSet, RBAC, Operator) |
| `deploy/helm/` | Helm charts |
| `config/` | Kubebuilder / kustomize scaffold |
| `scripts/` | Dev and cluster helper scripts |

## Tech stack

- **Language:** Go 1.22+
- **K8s:** `client-go` or kubebuilder / controller-runtime
- **Observability:** NVIDIA DCGM, Prometheus, alerting (e.g. XID errors)
- **Storage:** NFS / Ceph checkpoints for resume training

## Implementation order (from spec)

1. Go + `client-go`: cordon node, label/taint APIs.
2. Service reading Prometheus (or mock GPU fault metrics).
3. On fault: trigger healing — cordon → taint → evict pods → Job reschedules on healthy nodes.

## Git 工作流

- **开发分支：** `dev`（所有功能开发与 Agent 改代码默认在此分支）
- **稳定分支：** `main`（不直接开发；合并由用户决定）
- **提交信息：** 使用中文，说明改动目的

```bash
git checkout dev
```

## Conventions

- Prefer small, testable packages under `internal/`.
- Keep RBAC and manifests in `deploy/` in sync with operator permissions.
- Do not commit secrets; use `.env.example` for local config names only.
- Comments and user-facing docs may be Chinese; identifiers and code comments in English unless the file is already Chinese.

## Commands

```bash
make help          # list targets
make build         # build binaries
make test          # unit tests
make manifests     # generate / apply manifests (when wired)
```

## When changing behavior

- Update `docs/architecture.md` if the healing pipeline steps change.
- Update `deploy/manifests/` RBAC when new K8s API verbs are used.
