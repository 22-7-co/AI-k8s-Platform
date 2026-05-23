# P2 验收分级说明

> 更新：2026-05-21  
> 关联：[项目计划.md](../项目计划.md) §8 P2、ADR-0001

## 结论（面试官 / 自测口径）

| 问题 | 答案 |
|------|------|
| P2 做完了吗？ | **在 Plan B 前提下：是**（代码 + 真 K8s API + 本地 Operator 闭环已复验） |
| 与计划「完整 L1」一致吗？ | **尚未完全一致**；缺口列入 P3 硬化 |

## 两级验收

### L1-A：Plan B（当前已达成）

**环境：** 单节点 k3s（或任意 1-node 集群）  
**脚本：** `./scripts/e2e-k3s.sh`  
**故障输入：** `PROMETHEUS_MOCK=true` + `PROMETHEUS_MOCK_NODES=<fault-node>`  
**Operator：** 本地 `go run ./cmd/operator`（非 Deployment）  
**恢复语义：** Cordon → Taint → Evict → `WaitForReschedule` → Job **新 Pod Running**（允许同节点：驱逐后 uncordon + 去污点）

**已验证：**

- healing / prometheus / controller 单测 ≥60%
- 日志含 `action_id`、cordon/taint/evict/verify
- Node Events（Healing*）
- 脚本可重复跑（开头 delete Job）

### L1-B：完整 L1（计划原文，P3 目标）

| 项 | 要求 |
|----|------|
| 换节点 | 新 Pod `Running` 且 `spec.nodeName != fault-node` |
| e2e | `./scripts/e2e-kind.sh` 全绿（kind 2 worker） |
| 感知 | Exporter → Prometheus → Operator **真实 PromQL**（非 mock） |
| 部署 | Operator **集群内 Deployment** + SA/RBAC |
| 确认重建 | `WaitForReschedule` 轮询（已与 Informer 等价，不强制 Watch） |

## 与 ADR / 计划的关系

- **ADR-0001：** 控制面入口仍为 PromQL Pull；L1-B 仅把 mock 换成真实 Prometheus scrape。  
- **项目计划 §8 P2-7：** Gate 原文为 `e2e-kind.sh`；当前以 **e2e-k3s = L1-A Gate** 标记 P2 完成，**e2e-kind = L1-B**，归入 P3。

## P3 待办（由 P2 缺口转入）

1. `e2e-kind.sh` 在 CI / 本地 Docker 可用时跑通（2 节点）  
2. e2e 子路径：Exporter + Prometheus + 集群内 Operator  
3. `evictOnePod`：仅 Eviction 失败时 Delete（P3-0 已修）  
4. e2e 纳入 GitHub Actions（可选 `continue-on-error` → required）
