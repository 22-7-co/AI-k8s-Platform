# 面试 FAQ

> 高频追问标准答案（≤150 字/题）。配合 [interview-pitch.md](./interview-pitch.md)、[known-limitations.md](./known-limitations.md)。

## 架构选型

### Q1：为什么 PromQL Pull，不用 Alertmanager Webhook？

MVP 要 kind 单命令 e2e、healing/prometheus 可单测。Pull + mock 无需 AM、Ingress。Webhook 秒级但引入分组/抑制与重复触发归属，留 Post-MVP ADR。见 [adr/0001-mvp-promql-pull-only.md](./adr/0001-mvp-promql-pull-only.md)。

### Q2：为什么单副本 Operator 可以接受？

MVP 接受控制面 SPOF：Deployment 重启 + `operator_up` 告警。工作流幂等写在 Node 标签 `healing-state`（etcd），非内存；进程挂掉重启后读标签续跑。生产加 Leader Election。见 `internal/healing/orchestrator.go`。

### Q3：为什么 kind 不够，还要上云 VM？

kind 2 worker 能证 L1-B（`new_pod.node != fault_node`），但在 Docker 里，面试观感像 CI 玩具。3 台云 VM 有真实 hostname、VPC、Exporter DaemonSet 跨节点——主打录屏用 L3。kind 仍作 CI 回归。

### Q4：为什么没有 GPU / 真实 DCGM？

编排链路与指标源解耦：Exporter 可换官方 DCGM Exporter，Operator 只消费 PromQL。GPU 云主机成本高、驱动复杂；CPU + mock XID 已验证 **感知 → Cordon → Evict → Job 重建** 闭环。

## 可靠性与幂等

### Q5：`healing-state` 崩溃续跑怎么工作？

每步 patch Node 标签：空 → cordoned → tainted → evicted → completed。Operator OOM 重启后 `AdvanceHealing` 读标签跳过已完成步骤，类似粗粒度 saga。见 `internal/healing/state.go`、`docs/runbook-uncordon.md`。

### Q6：Operator 宕机窗口内会怎样？

新 XID 故障在重启完成前不会被 Pull 处理；进行中的 healing 重启后续跑。MVP 靠 `operator_up` + 外部告警；不承诺控制面 HA。

### Q7：30s 轮询 vs「≤10s Cordon」矛盾吗？

**≤10s** 指 PromQL 命中后 **Cordon+Taint 执行**（e2e 实测 P99 约数秒）。端到端含 scrape + 轮询周期，demo 口径 **≤90s**（见 `interview-pitch.md` 亮点数据）。

## 训练与工作负载

### Q8：batch Job vs PyTorchJob / gang 调度？

MVP 用 `batch/v1` Job 降复杂度，证 **单 Pod 训练任务** 可迁节点。分布式 gang 需 Volcano 适配器：故障节点上按 label 批量 evict 整组 Pod——Post-MVP，见 [interview-deep-dive.md](./interview-deep-dive.md)。

### Q9：NCCL hang 和 XID 谁先处理？

本平台 MVP 针对 **XID 等硬件 counter 信号**。NCCL hang 常晚于 XID，且需框架侧 timeout；Post-MVP 可加训练侧 heartbeat 指标合取，避免单源误报。

### Q10：续训谁负责？

Operator **不**实现梯度续训。L2 示例 Job 挂载 PVC + 约定 checkpoint 路径；读 checkpoint 在训练镜像。平台只保证 Pod 重建与存储挂载契约。

## 隔离与驱逐

### Q11：为什么整节点 Cordon，不是单卡？

MVP 节点级隔离：一张卡 XID 常导致 NCCL 集体失败。部分 GPU / MIG 故障粒度留 Post-MVP；面试主动说明简化。

### Q12：Eviction 失败怎么办？

默认 `policy/v1` Eviction；API 拒绝或超时 fallback `Delete`。PDB 可能导致 evict err——metrics 有 `result=err`，需人工或调 PDB。见 `internal/healing/evict.go`。

## 竞品分工

### Q13：和 NPD / Cluster Autoscaler / 官方 DCGM Exporter 的区别？

| 方案 | 缺口 |
|------|------|
| DCGM Exporter | 只暴露指标，不驱逐 |
| NPD | 通用 Node Condition，不覆盖 XID→训练 Pod 闭环 |
| CA | 扩容，不管已 Cordon 节点上 Pod 迁移 |
| **本项目** | Prometheus 信号 + healing 状态机 + Job 级恢复 |

## 演示与环境

### Q14：L1-A 单节点 e2e 能当主打 demo 吗？

**不能。** L1-A（`e2e-k3s.sh`）是 Plan B 开发自测：驱逐后 uncordon 同节点恢复。面试主打 **L1-B kind** 或 **L3 云 VM**（严格换节点）。

### Q15：云上 lab 怎么复现？

见 [cloud-lab.md](./cloud-lab.md)。3× ecs.e-c1m2.large k3s，录屏短租 2–4h，`./scripts/demo-cloud.sh`。
