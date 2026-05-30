# Grafana 示例（P5-Obs）

仅用于**本地演示 / 录屏**，非生产 Helm 部署。

## 内容

| 路径 | 说明 |
|------|------|
| `provisioning/datasources/prometheus.yml` | 数据源模板（`start-grafana.sh` 渲染 URL） |
| `provisioning/dashboards/dashboard.yml` | Dashboard 文件 Provider |
| `dashboards/healing-overview.json` | **AI 平台自愈监控**（UID `ai-k8s-healing`） |

## 启动

```bash
./scripts/observability-stack.sh up
# 或单独
./scripts/grafana/start-grafana.sh up
```

浏览器：<http://localhost:3000>，账号 **admin / admin**（勿暴露公网）。

## 重新导出 Dashboard JSON

1. Grafana UI → Dashboard **AI 平台自愈监控** → Share → Export → Save JSON  
2. 覆盖 `dashboards/healing-overview.json`  
3. 保持 `uid: ai-k8s-healing` 不变，以便 provisioning 稳定加载  

## 面板（v3）

| 面板 | 指标 |
|------|------|
| Operator 运行状态 | `operator_up` |
| 自愈动作速率 | `healing_actions_total` rate |
| 距上次自愈成功 | `time() - healing_last_success_timestamp` |
| GPU XID 错误 | `gpu_xid_errors_total`（支持 `$node`） |
| 自愈耗时 P99 | `healing_duration_seconds` |
| 自愈失败次数 | `healing_actions_total{result!="ok"}` |
| 恢复确认速率 | `healing_recovery_total` |

录屏：见 [docs/demo-runbook.md](../../demo-runbook.md) 模式四、`scripts/demo-record.sh`。
