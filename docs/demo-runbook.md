# 演示 Runbook

> 主演示脚本：`../scripts/demo.sh`  
> 面试话术：[interview-pitch.md](./interview-pitch.md)

## 前置

| 依赖 | L1-A（默认） | L1-B（`--kind`） |
|------|----------------|------------------|
| Go 1.22+ | ✅ | ✅ |
| kubectl + 集群 | k3s / 任意 1+ 节点 | — |
| Docker + kind | — | ✅ |

```bash
git checkout dev
make build
go test ./... -count=1   # 可选
```

**Context 提示：** 跑完 kind 后执行 `kubectl config use-context default`（或你的 k3s context）再跑 L1-A。

## 模式一：L1-A（推荐现场，~1 分钟）

```bash
kubectl config use-context default   # 示例：k3s
./scripts/demo.sh
```

脚本会：

1. apply RBAC / 训练 Job  
2. mock PromQL 指向当前 Pod 所在节点  
3. 本地启动 Operator（metrics 默认 `:18081`，避免与已有进程抢 `:8080`）  
4. 等待 cordon → 驱逐 → 新 Pod Running  
5. 打印 nodes / pods 摘要  
6. 提示 `./scripts/uncordon.sh <node>`

单节点集群会自动 uncordon 故障节点（Plan B），见 [p2-acceptance.md](./p2-acceptance.md)。

## 模式二：L1-B（双节点换节点，~2–3 分钟）

```bash
./scripts/demo.sh --kind
```

等价于 `./scripts/e2e-kind.sh`：创建 kind 2-worker、镜像 preload、集群内 Operator Deployment、断言新 Pod 在**另一 worker**。

可选真 PromQL 子路径：

```bash
KEEP_CLUSTER=true RUN_PROMQL_E2E=true ./scripts/e2e-kind.sh
```

## 模式三：Dry-run（不改集群）

```bash
./scripts/demo.sh --dry-run
```

- `HEALING_DRY_RUN=true`：打 JSON 日志，不 Cordon / 不驱逐 / 不写 Event  
- 验证节点仍为可调度  

适合讲解状态机而不动生产/共享集群。

## 模式四：Grafana 联调（P5-Obs）

**依赖：** Docker；宿主机已 `make build`。

| 终端 | 命令 |
|------|------|
| T1 | `./bin/exporter &` |
| T1 | `METRICS_LISTEN=:18081 go run ./cmd/operator &`（按需设 `PROMETHEUS_MOCK_*`） |
| T2 | `./scripts/observability-stack.sh up` |
| T3 | `./scripts/demo.sh` 或 `curl -X POST 'localhost:9100/inject/xid?node=<node>'` |

打开 <http://localhost:3000> → Dashboard **AI Platform Healing**（`operator_up`、healing 曲线、XID 面板）。

与 `demo.sh` 并行即可，**不必**改 demo 脚本逻辑；结束执行 `./scripts/observability-stack.sh down`。

## 演示后回滚

```bash
./scripts/uncordon.sh <fault-node>
kubectl get node <fault-node> -o yaml | grep -E 'unschedulable|healing-state|gpu-fault'
```

审批与误报流程见 [runbook-uncordon.md](./runbook-uncordon.md)。

## L2 可选说明（口播即可）

示例 Job 可挂载 PVC，容器启动脚本读取约定 checkpoint 路径（演示可用 `touch`）。**续训逻辑在训练框架**，Operator 只保证 Pod 重建与存储挂载契约——非平台 MVP 能力。

## 故障排查

| 现象 | 检查 |
|------|------|
| `kubectl cannot reach a cluster` | `kubectl cluster-info`、context |
| `:8080 bind: address already in use` | 用 `demo.sh`（默认 `:18081`）或停旧 Operator |
| kind 镜像拉取失败 | `docker info`、代理 / `kind load` |
| 新 Pod 仍在 fault 节点 | 需 L1-B 双节点；单节点走 Plan B |

## 相关脚本

| 脚本 | 用途 |
|------|------|
| `scripts/e2e-k3s.sh` | L1-A Gate |
| `scripts/e2e-kind.sh` | L1-B Gate |
| `scripts/e2e-promql.sh` | 真 PromQL 子 Gate |
| `scripts/uncordon.sh` | 人工回滚 |
