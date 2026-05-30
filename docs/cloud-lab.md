# L3 云上真实集群 Lab（录屏短租）

> **选型：** 阿里云 3× `ecs.e-c1m2.large`（2C4G）按量付费，Debian 13，华东 1。  
> **用途：** 面试主打 demo；kind 仍作 CI（L1-B）。

## 拓扑

| VM | 角色 | 组件 |
|----|------|------|
| ai-k8s-control | k3s server | Operator、Prometheus Deployment |
| ai-k8s-worker-a | k3s agent | Exporter DaemonSet、训练 Pod 初始 |
| ai-k8s-worker-b | k3s agent | Exporter DaemonSet、故障后 Pod 目标 |

Grafana（可选）：control 宿主机 Docker，`./scripts/observability-stack.sh up` 指向集群 Prometheus NodePort/端口转发。

## 安全组（开机器后第一件事）

| 方向 | 端口 | 用途 |
|------|------|------|
| VPC 内全放通 | 6443/TCP, 8472/UDP, 10250/TCP | k3s |
| 本机公网 IP | 22/TCP | SSH |
| 本机公网 IP | 6443/TCP | kubectl（或 SSH 隧道） |
| 本机公网 IP | 9100/TCP | Exporter inject（hostNetwork） |
| 本机公网 IP | 3000/TCP | Grafana 录屏（若开） |

## 安装 k3s

**control：**

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --disable traefik --write-kubeconfig-mode 644" sh -
sudo cat /var/lib/rancher/k3s/server/node-token   # 给 worker 用
sudo cat /etc/rancher/k3s/k3s.yaml                # 合并到本机 ~/.kube/config
```

**每个 worker：**

```bash
export K3S_URL=https://<control-private-ip>:6443
export K3S_TOKEN=<token>
curl -sfL https://get.k3s.io | sh -
```

或使用 [`scripts/cloud/install-k3s-agent.sh`](../../scripts/cloud/install-k3s-agent.sh)。

**验证：**

```bash
kubectl get nodes -o wide   # 3 Ready
```

## 部署平台栈

在本机（已配置 KUBECONFIG 指向云集群）：

```bash
make build
./scripts/cloud/deploy-stack.sh
```

构建并加载镜像（若 worker 无法拉私有镜像，在 control 上 `docker save | ssh worker ctr import`，或改用 registry）。

## 演示与 e2e

```bash
# 完整 e2e（真 PromQL + inject XID + 换节点）
./scripts/e2e-cloud.sh

# 面试录屏入口
./scripts/demo-cloud.sh
```

## 故障注入

Exporter 使用 `hostNetwork:9100`：

```bash
FAULT_NODE=ai-k8s-worker-a
IP=$(kubectl get node "$FAULT_NODE" -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}')
curl -sf -X POST "http://${IP}:9100/inject/xid?node=${FAULT_NODE}&gpu_id=0&xid_code=79"
```

若本机无法直连内网 IP，在 control 上执行上述 curl，或 SSH 到 worker。

## 录屏日时间线（约 90 分钟）

1. 开 3 台 → 安全组 → 装 k3s（30 min）
2. `./scripts/cloud/deploy-stack.sh` + `./scripts/demo-cloud.sh` 彩排
3. Grafana Last 15m + `kubectl get pods -o wide -n ai-training` 录 2–3 min
4. **释放 3 台实例**

## 成本

3 台按量 × 2–4h ≈ **1–3 元**（以控制台为准）。录完即释放。

## 与 kind 差异

见 [interview-faq.md](./interview-faq.md) Q3、[known-limitations.md](./known-limitations.md)。

## 回滚

```bash
./scripts/uncordon.sh <fault-node>
```
