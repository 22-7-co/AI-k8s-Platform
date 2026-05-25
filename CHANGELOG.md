# Changelog

本仓库遵循 [项目计划.md](./项目计划.md) 阶段 Gate；用户可见变更集中记录于此（P4 起维护）。

## [Unreleased]

### Added

- **P4：** `scripts/demo.sh`（L1-A / `--kind` / `--dry-run`）、`docs/interview-pitch.md`、`docs/demo-runbook.md`
- **P3：** Node Update 冲突重试（`updateNodeWithRetry`）、Operator `/metrics`、`HEALING_MAX_RETRIES` 指数退避、`scripts/uncordon.sh` 与 `docs/runbook-uncordon.md`
- **P3 L1-B：** `scripts/e2e-kind.sh`（2 worker + 集群内 Deployment）、`scripts/e2e-promql.sh`（真 PromQL 子路径）
- **P2：** Operator PromQL 轮询闭环、cordon/taint/evict、`scripts/e2e-k3s.sh`（L1-A Plan B）
- **P1：** mock Exporter、Prometheus 客户端、`scripts/prometheus/start-prometheus.sh`
- **P0：** `internal/healing`（Cordon、污点、状态机）、RBAC 清单

### Changed

- Operator 日志改为 **JSON**（`action_id`、`promql`、`dry_run` 等）
- CI：`e2e-kind` job 在 `dev`/`main` 触发；P4 起 **required**（不再 `continue-on-error`）
- Eviction API 优先，仅失败时 fallback Delete

### Fixed

- Cordon 与 Taint 连续 patch 同一 Node 时的 **resourceVersion Conflict**（e2e 日志不再出现 `the object has been modified`）

## 版本说明

尚未打 tag；合并 `main` 后可发布 `v0.1.0-demo`（可选）。
