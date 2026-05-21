# internal/healing

Self-healing orchestration:

- Cordon node
- Add taints
- Evict or delete affected pods
- Optional dry-run via `HEALING_DRY_RUN`
