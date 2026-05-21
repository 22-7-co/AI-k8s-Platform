# internal/controller

Kubernetes reconcilers and watches (Pods, Nodes, optional CRDs).

Planned responsibilities:

- Subscribe to workload events relevant to training jobs
- Delegate fault response to `internal/healing`
