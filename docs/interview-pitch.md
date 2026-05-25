# 面试话术（5 分钟版）

> 对齐 [项目计划.md](../项目计划.md) §9、§14；验收分级见 [p2-acceptance.md](./p2-acceptance.md)。

## 一句话

**机器冒烟前把任务捞出来，机器宕机后让任务自动复活。**

## 架构（30 秒）

```mermaid
flowchart LR
  Exporter[Go Exporter / mock XID] --> Prom[Prometheus]
  Prom --> Op[Operator PromQL Pull]
  Op --> H[Cordon → Taint → Evict]
  H --> Job[batch/v1 Job 重建 Pod]
```

- **感知：** `gpu_xid_errors_total`（MVP 可 mock 注入；Post-MVP 接 DCGM Exporter）。
- **决策与执行：** 专用 Operator + `healing-state` 状态机（非 Alertmanager Webhook）。
- **恢复（K8s 层）：** Job 控制器在健康节点拉起新 Pod；`WaitForReschedule` 确认 Running。

## 与社区方案差异（简表）

| 方案 | 做什么 | 与本项目 |
|------|--------|----------|
| NVIDIA DCGM Exporter | GPU 指标 | MVP 自研 mock；**可替换**，Operator 不重复采集 |
| Node Problem Detector | 节点 Condition | 通用节点故障；**不**覆盖 XID→训练驱逐闭环 |
| Cluster Autoscaler | 扩容 | 不管已 Cordon 节点上的 Pod 迁移 |
| GPU Operator | 驱动 / Device Plugin | 集群底座，与本项目互补 |
| Volcano / Kueue | 队列 / gang | MVP 用 `batch/v1` Job |
| 告警 + 人工 runbook | 通知 | 无自动 Cordon/驱逐；我们是 **closed-loop** |

**一句：** Exporter 可换官方；核心是 **Prometheus 信号 + healing 状态机 + Job 级恢复**，NPD/CA 解决不了训练 Pod 自动迁走。

## MVP 边界

| 层级 | 平台承诺 | 不承诺 |
|------|----------|--------|
| **L1** | PromQL 发现故障 → 隔离节点 → 驱逐带标签训练 Pod → Job 新 Pod Running | 梯度续训、NCCL 容错 |
| **L2（演示加分）** | 示例 Job + PVC 挂载 checkpoint 路径（可 touch 文件） | Operator 内实现续训逻辑 |

**表述：** Operator 保证**调度侧自愈**；续训是训练镜像 + Checkpoint 契约。

## 亮点数据（e2e 口径）

| 口径 | 脚本 | 说明 |
|------|------|------|
| **L1-A Plan B** | `./scripts/e2e-k3s.sh` | 单节点 k3s；驱逐后 uncordon，新 Pod 可同节点 Running |
| **L1-B** | `./scripts/e2e-kind.sh` | kind 2 worker；`fault-node != new-pod.nodeName`；集群内 Deployment |
| **真 PromQL** | `RUN_PROMQL_E2E=true` + `e2e-promql.sh` | Exporter inject → Prometheus instant query → Operator cordon |

本地验证：2026-05-25（P3 Gate + P4 文档）。

## 排障

| 场景 | 做法 |
|------|------|
| 误隔离 | `./scripts/uncordon.sh <node>`（见 [runbook-uncordon.md](./runbook-uncordon.md)） |
| 只看日志、不改集群 | `HEALING_DRY_RUN=true` 或 `./scripts/demo.sh --dry-run` |
| 流程卡在哪 | `kubectl get node <n> -o yaml` 看 `healing-state`：空 → cordoned → tainted → evicted → completed |
| 指标 | Operator `:8080/metrics`（`healing_actions_total`、`operator_up`） |

## 演进（Post-MVP 一句）

Alertmanager Webhook、HealingPolicy CRD、Leader Election、官方 DCGM 对接、Volcano Job——见计划 §15。

## 演示命令

```bash
make build && ./scripts/demo.sh           # L1-A（当前 kubectl context）
./scripts/demo.sh --kind                  # L1-B（需 Docker + kind）
./scripts/demo.sh --dry-run               # 仅 JSON 日志，零 K8s 写
```

详细步骤：[demo-runbook.md](./demo-runbook.md)。
