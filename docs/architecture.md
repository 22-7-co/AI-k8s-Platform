# Architecture

## Problem

Large-model training runs for weeks on hundreds of GPUs. A single GPU fault (thermal, ECC, XID) can break NCCL and kill the whole job. SREs need **automated fault tolerance**, not manual restarts at 3am.

## Layers

### 1. Perception (Exporter + Prometheus)

- **MVP:** Go **exporter** (`cmd/exporter`) exposes `gpu_xid_errors_total`; fault injection via `POST /inject/xid?node=...` for e2e.
- **Post-MVP:** NVIDIA DCGM DaemonSet + official DCGM Exporter (replace or complement mock).
- Prometheus scrapes exporter; **no Alertmanager** in MVP control path (ADR-0001).

### 2. Control plane (Operator)

- **Entry:** PromQL **pull** on a timer (`POLL_INTERVAL`, default 30s).
- **Mock path:** `PROMETHEUS_MOCK=true` + `PROMETHEUS_MOCK_NODES` for L1-A / fast e2e.
- **Real path:** `PROMETHEUS_URL` + instant/range query (e.g. `gpu_xid_errors_total > 0`).
- Binary: `cmd/operator` + `internal/operator` loop; cluster deploy via `deploy/manifests/operator/`.

### 3. Self-healing pipeline (`internal/healing`)

| Step | Action | Node label `healing-state` |
|------|--------|----------------------------|
| 1 | Cordon (`Unschedulable=true`) | `cordoned` |
| 2 | GPU fault NoSchedule taint | `tainted` |
| 3 | Evict training pods (Eviction API, Delete fallback) | `evicted` |
| 4 | `WaitForReschedule` — Job pod Running on healthy node | — |
| 5 | Mark completed + cooldown annotation | `completed` |

**Idempotency:** label state machine + `HEALING_COOLDOWN`; Node updates use **Get → mutate → Update with conflict retry** (max 3).

**Dry-run:** `HEALING_DRY_RUN=true` logs all steps, no K8s writes, no Events, metrics skipped.

**Failure:** exponential backoff (`HEALING_MAX_RETRIES`, default 5), `healing-fail-count` annotation.

### 4. Workload recovery (Kubernetes)

- `batch/v1` Job controller schedules a new Pod (L1-B: **different** node; L1-A Plan B: same node after uncordon).
- **L2 (demo only):** example Job + PVC checkpoint path — resume logic in training image, not Operator.

## Observability

| Surface | Content |
|---------|---------|
| Logs | JSON (`action_id`, `node`, `promql`, `action`, `dry_run`, `result`, `error`) |
| Metrics | `:8080/metrics` — `operator_up`, `healing_actions_total`, `healing_duration_seconds`, `healing_last_success_timestamp` |
| Events | `HealingCordon` / `HealingTaint` / … on Node (skipped in dry-run) |

## Acceptance paths

| Tier | Script | Notes |
|------|--------|-------|
| L1-A | `scripts/e2e-k3s.sh` | Single-node Plan B, local `go run` operator |
| L1-B | `scripts/e2e-kind.sh` | kind 2 workers, in-cluster Deployment |
| PromQL | `scripts/e2e-promql.sh` | Host exporter + Prometheus + real query |
| Demo | `scripts/demo.sh` | Wraps e2e / dry-run for presentations |

## Interview narrative

> Built a closed-loop Kubernetes Operator: **Prometheus signal → dedicated healing state machine → Job-level recovery**. Platform delivers scheduling-side self-healing; checkpoint resume is a training/PVC contract (L2), not Operator magic.

## Related docs

- [docs/adr/0001-mvp-promql-pull-only.md](./adr/0001-mvp-promql-pull-only.md)
- [docs/p2-acceptance.md](./p2-acceptance.md) — L1-A / L1-B grading
- [docs/interview-pitch.md](./interview-pitch.md)
- [docs/demo-runbook.md](./demo-runbook.md)
- [AGENTS.md](../AGENTS.md)
