# Runbook: 节点解除隔离（uncordon）

## 何时使用

- **误报**：PromQL/Exporter 注入错误，节点被错误 cordon/taint。
- **人工恢复**：硬件维修完成，SRE 批准将节点重新纳入调度。
- **演练结束**：e2e/压测后清理节点状态。

## 前置审批

| 角色 | 动作 |
|------|------|
| 值班 SRE | 确认该节点无真实 GPU XID/硬件告警 |
| 集群负责人 | 生产环境需书面/工单批准（记录工单号到 Event 备注） |

## 操作步骤

```bash
./scripts/uncordon.sh <node-name>
```

脚本会：

1. `kubectl uncordon`
2. 移除 `ai-k8s-platform.io/gpu-fault` NoSchedule 污点
3. 清除 `healing-state`、`healing-completed-at`、`healing-fail-count` 注解

## 验证

```bash
kubectl get node <node> -o yaml | grep -E 'unschedulable|healing-state|gpu-fault'
kubectl get pods -A --field-selector spec.nodeName=<node>
```

节点应 `unschedulable: false`，无 healing-state=completed/cordoned 残留（除非仍有进行中的自愈）。

## 与 Operator 的关系

- 冷却期内（`healing-state=completed` + `healing-completed-at`）Operator **不会**再次处理该节点。
- uncordon 后若 Prometheus 仍报障，下一轮 poll 会重新触发自愈；务必先修复指标源。

## 回滚误操作

若 uncordon 过早且故障仍存在，可手动：

```bash
kubectl cordon <node>
kubectl taint nodes <node> ai-k8s-platform.io/gpu-fault:NoSchedule
```

并通知 Operator 负责人检查 `healing-fail-count` 与 `/metrics` 中 `healing_actions_total`。
