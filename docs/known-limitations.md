# 已知简化与边界（MVP）

> 面试**主动说明**下表，比被追问击穿更有说服力。Post-MVP 路线见 [项目计划.md](../项目计划.md) §15。

| 简化项 | MVP 现状 | 生产需补 | 面试怎么说 |
|--------|----------|----------|------------|
| 控制面 HA | 单副本 Operator | Leader Election + lease | 进程无 HA，工作流靠 Node 标签幂等 |
| 故障输入 | PromQL 轮询（默认 30s） | Alertmanager Webhook、抑制/分组 | MVP 求可测；延迟下限=轮询周期 |
| 指标源 | mock XID Exporter | 官方 DCGM Exporter | 指标源可替换，编排不变 |
| 工作负载 | `batch/v1` 单 Pod Job | Volcano / PyTorchJob gang | 证调度侧自愈，非 NCCL 容错 |
| 续训 | L2 PVC 示例（可选） | 训练框架 + 存储 | Operator 不实现梯度续训 |
| 隔离粒度 | 整节点 Cordon+Taint | 单卡 / MIG 策略 | 节点级与 XID 扩散模型一致 |
| RBAC | ClusterRole + 选择器收窄 | Per-NS SA、Admission 审批 | dry-run + 冷却 + 人工 uncordon |
| L1-A 单节点 | 同节点恢复（Plan B） | — | **非主打**；CI/开发用，demo 用 L1-B/L3 |
| kind 集群 | Docker 内 2 worker | 云 VM 多节点 | kind=CI；录屏用 L3 云 lab |
| GPU | 无 | GPU 云 + 驱动 | CPU mock 验证闭环，GPU 换 Exporter |
| 误报 | 冷却 + healing-state | 多信号合取、flapping 窗口 | 单次 XID spike 可能 cordon，靠 uncordon 回滚 |
| Operator 自愈 | 无（靠 Deployment 重启） | 多副本 + 选主 | `operator_up` 外部告警 |

## 三级演示体系

| 级别 | 环境 | 用途 |
|------|------|------|
| L1 | k3s / kind | CI、`make test`、开发 Gate |
| L2 | 本机 Grafana + PVC 示例 | 指标与 checkpoint 契约演示 |
| L3 | 3 台云 VM k3s | **面试主打录屏** |

## 不承诺（Out of Scope）

- 替代 Kubeflow / Volcano / Slurm 全套调度
- NCCL 容错、In-job 梯度续训逻辑
- 多集群联邦、零停机 Operator 升级
- 生产级 Grafana 告警值班全套（有示例 Rule，见 `docs/examples/prometheus-rule.yaml`）
