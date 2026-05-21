# ADR-0001：MVP 控制面仅使用 PromQL 定时轮询

- **状态：** 已接受  
- **日期：** 2026-05-21  
- **决策者：** 项目维护者  
- **关联：** [项目计划.md §3.0](../../项目计划.md)

## 背景

自愈 Operator 需要知道「哪张节点 GPU 故障」。可选输入包括：Prometheus Alertmanager Webhook、Operator 内 PromQL 轮询、Informer 监听 Node Condition（如 NPD）、Informer 监听 Pod 状态。

MVP 需在可演示、可单测、依赖最少的前提下选定**唯一**故障发现与触发路径。

## 决策

**MVP 仅采用 PromQL 定时轮询作为控制面输入；幂等由 Node 标签 `ai-k8s-platform.io/healing-state` + 冷却时间负责。**

### 模式对照

| 模式 | MVP | 负责模块 | 延迟 | 重复触发 | 幂等 |
|------|-----|----------|------|----------|------|
| **PromQL 定时轮询** | ✅ **唯一入口** | `internal/prometheus` + `internal/healing` | scrape + 轮询周期（默认 30s） | 同节点可多次命中 | `healing-state` 标签 + `HEALING_COOLDOWN` |
| Alertmanager Webhook | ❌ Post-MVP | - | 秒级 | 需 AM 分组/抑制设计 | 同上 + Webhook handler |
| Informer → Node Condition | ❌ 非故障发现 | `internal/controller` | 实时 | 与 XID 不同源 | 不触发隔离 |
| Informer → Pod | ⚠️ P2 辅助 | `internal/controller` | 实时 | Watch 重放 | **仅**确认 Job 重建，不触发驱逐 |

### P2 验收路径

```
Exporter(mock) → Prometheus → Operator PromQL 轮询 → healing API → Job 重建
```

**不依赖** Alertmanager、CRD、NPD Condition 作为触发源。

### 为何不三种并行

- MVP 目标：kind 上单命令 e2e、healing/prometheus 包可单测。  
- Pull + mock 指标无需 AM、Ingress、额外 CRD。  
- Webhook / 多输入源会引入重复触发与幂等归属争议，留 Post-MVP 新 ADR。

## 后果

### 正面

- 架构单一，面试叙事清晰。  
- `internal/prometheus` 与 `internal/healing` 边界明确。  
- 本地开发仅需 Prometheus + mock Exporter。

### 负面 / 权衡

- 故障感知延迟下限为轮询周期（非 Webhook 秒级）。  
- Alertmanager 抑制/分组能力 MVP 不可用。  
- GPU XID 与 NPD Condition 未统一（Post-MVP 可 ADR-0002 补充）。

## 关联决策（未在本 ADR 范围）

- 驱逐：默认 `policy/v1` Eviction，失败 fallback Delete（见项目计划 §7.1、P2-2）。  
- 进程崩溃恢复：仅 `healing-state` Node 标签断点，不用内存态幂等。  
- Plan B 排期：可先 Cordon + 驱逐，Taint / Pod Informer 后置（见项目计划 §8 P2 脚注）。

## 状态变更

| 日期 | 状态 |
|------|------|
| 2026-05-21 | 已接受 |
