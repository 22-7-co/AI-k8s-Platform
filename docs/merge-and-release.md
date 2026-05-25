# 合并与发布备忘（P0–P4）

> PR #1 已合并 `dev` → `main` 时，本文档供后续小版本 PR 或 release 参考。

## PR 标题示例

```
feat: MVP 自愈平台 P0–P4（L1-A/L1-B + CI e2e-kind）
```

## PR 正文（可粘贴）

### Summary

- **P0–P1：** healing 状态机、RBAC、mock Exporter、PromQL 客户端
- **P2 L1-A：** mock PromQL → Cordon/Taint/Evict → Job 新 Pod Running（`e2e-k3s.sh`，单节点 Plan B）
- **P3 L1-B：** kind 双节点换节点、集群内 Operator、Conflict 重试、metrics、JSON 日志、退避、uncordon
- **P4：** `demo.sh`、`interview-pitch`、CHANGELOG；CI `e2e-kind` required

### E2E（§4.2 口径）

| 级别 | 脚本 | 说明 |
|------|------|------|
| L1-A | `./scripts/e2e-k3s.sh` / `./scripts/demo.sh` | 单节点 Plan B |
| L1-B | `./scripts/e2e-kind.sh` / `./scripts/demo.sh --kind` | 严格换节点 |
| 真 PromQL | `RUN_PROMQL_E2E=true` + `e2e-promql.sh` | 可选子 Gate |

### 演示与回滚

```bash
./scripts/demo.sh
./scripts/demo.sh --kind
./scripts/uncordon.sh <node>
```

### Test plan

- [x] `go test ./...` + `make build`
- [x] GitHub Actions `e2e-kind` 绿

---

## 合并后（可选）

```bash
git checkout main && git pull origin main
git tag -a v0.1.0-demo -m "MVP demo: L1-A/L1-B self-healing platform"
git push origin v0.1.0-demo
```

`CHANGELOG.md` 中 `[0.1.0-demo]` 已与上述 tag 对齐。

## 环境清理（可选）

```bash
kind delete cluster --name ai-k8s-e2e
kubectl config use-context default   # 切回 k3s
```
