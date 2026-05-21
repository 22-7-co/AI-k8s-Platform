# Architecture

## Problem

Large-model training runs for weeks on hundreds of GPUs. A single A100 fault (thermal, ECC, XID) can break NCCL and kill the whole job. SREs need **automated fault tolerance**, not manual restarts at 3am.

## Layers

### 1. Perception (DCGM + Exporter)

- DaemonSet on each GPU node runs NVIDIA DCGM.
- Go **exporter** (`cmd/exporter`) exposes hardware metrics to Prometheus.

### 2. Control plane (Operator)

- Watches **Pod** lifecycle and **GPU-related metrics** (via Prometheus or cached alerts).
- Implemented in `cmd/operator` + `internal/controller`.

### 3. Self-healing pipeline

| Step | Action |
|------|--------|
| Warn | Prometheus fires on e.g. `XID 79` for `node-3` |
| Isolate | Operator cordons node, adds taint — no new schedules |
| Evict | Delete/evict training pods on the bad node |
| Resurrect | Job controller schedules on healthy nodes |
| Resume | New pod mounts latest checkpoint from NFS/Ceph |

## Code milestones (from product spec)

1. **Node APIs** — Go script/operator path: cordon, labels, taints via `client-go`.
2. **Metrics consumer** — Poll Prometheus (or mock) for GPU fault signals.
3. **Closed loop** — On fault → healing actions → verify pod rescheduled.

## Interview narrative

> Built a closed-loop Kubernetes Operator that ties GPU hardware metrics to scheduling: automatic node isolation and minute-level job recovery on shared checkpoints.

## Related docs

- [说明文档.txt](../说明文档.txt) — original product notes (Chinese)
- [AGENTS.md](../AGENTS.md) — repo map for Cursor agents
