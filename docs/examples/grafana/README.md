# Grafana 示例（P5-Obs）

仅用于**本地演示 / 录屏**，非生产 Helm 部署。

## 内容

| 路径 | 说明 |
|------|------|
| `provisioning/datasources/prometheus.yml` | 数据源模板（`start-grafana.sh` 渲染 URL） |
| `provisioning/dashboards/dashboard.yml` | Dashboard 文件 Provider |
| `dashboards/healing-overview.json` | **AI Platform Healing**（UID `ai-k8s-healing`） |

## 启动

```bash
./scripts/observability-stack.sh up
# 或单独
./scripts/grafana/start-grafana.sh up
```

浏览器：<http://localhost:3000>，账号 **admin / admin**（勿暴露公网）。

## 重新导出 Dashboard JSON

1. Grafana UI → Dashboard **AI Platform Healing** → Share → Export → Save JSON  
2. 覆盖 `dashboards/healing-overview.json`  
3. 保持 `uid: ai-k8s-healing` 不变，以便 provisioning 稳定加载  

## 面板与 PromQL

见 [docs/observability.md](../../observability.md) §4。
