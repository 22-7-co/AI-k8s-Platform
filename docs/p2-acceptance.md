# P2 / P3 验收分级说明

> 更新：2026-05-25  
> 关联：[项目计划.md](../项目计划.md) §8 P2、§8 P3、ADR-0001

## 结论（面试官 / 自测口径）

| 问题 | 答案 |
|------|------|
| P2 做完了吗？ | **是**（L1-A Plan B + 代码 Gate） |
| L1-B（严格换节点）做完了吗？ | **是**（P3 本地 + CI 路径已验证，2026-05-25） |
| P3 硬化做完了吗？ | **是**（Conflict 重试、metrics、JSON 日志、退避、uncordon、CI） |

## 两级验收

### L1-A：Plan B（已达成）

**环境：** 单节点 k3s（或任意 1-node 集群）  
**脚本：** `./scripts/e2e-k3s.sh` 或 `./scripts/demo.sh`  
**故障输入：** `PROMETHEUS_MOCK=true` + `PROMETHEUS_MOCK_NODES=<fault-node>`  
**Operator：** 本地 `go run ./cmd/operator`（非 Deployment）  
**恢复语义：** Cordon → Taint → Evict → `WaitForReschedule` → Job **新 Pod Running**（允许同节点：驱逐后 uncordon + 去污点）

**已验证：**

- healing / prometheus / controller 单测 ≥60%
- 日志含 `action_id`、cordon/taint/evict/verify
- 脚本可重复跑（开头 delete Job）
- **本地复验：** 2026-05-25，`e2e-k3s` PASS

### L1-B：完整 L1（已本地验证，2026-05-25）

| 项 | 要求 | 状态 |
|----|------|------|
| 换节点 | 新 Pod `Running` 且 `spec.nodeName != fault-node` | ✅ kind 2 worker |
| e2e | `./scripts/e2e-kind.sh` 全绿 | ✅ |
| 感知 | Exporter → Prometheus → Operator **真实 PromQL** | ✅ `e2e-promql.sh` / `RUN_PROMQL_E2E=true` |
| 部署 | Operator **集群内 Deployment** + SA/RBAC | ✅ |
| 确认重建 | `WaitForReschedule` 轮询 | ✅ |

**演示：** `./scripts/demo.sh --kind`

## 与 ADR / 计划的关系

- **ADR-0001：** 控制面入口仍为 PromQL Pull；L1-B 将 mock 换成真实 Prometheus scrape（子路径）。  
- **Gate 映射：** `e2e-k3s` = L1-A；`e2e-kind` = L1-B；P3 硬化不扩大 Operator 功能边界。

## P3 硬化（已完成）

| 项 | 说明 |
|----|------|
| Node Conflict 重试 | `internal/healing/node_retry.go` |
| `/metrics` | `internal/operator/metrics` |
| JSON 日志 | `internal/operator/logging` |
| 退避 + `healing-fail-count` | `HEALING_MAX_RETRIES` |
| uncordon | `scripts/uncordon.sh` + `docs/runbook-uncordon.md` |
| CI | `.github/workflows/ci.yml` — `e2e-kind`（P4 起 required） |
