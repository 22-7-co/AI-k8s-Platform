// Package metrics defines Prometheus metric and label names.
package metrics

const (
	// MetricGPUXIDErrorsTotal is the counter for GPU XID hardware errors.
	MetricGPUXIDErrorsTotal = "gpu_xid_errors_total"

	// LabelNode is the node name label on GPU metrics.
	LabelNode = "node"

	// LabelGPUID is the GPU index label.
	LabelGPUID = "gpu_id"

	// LabelXIDCode is the NVIDIA XID error code label.
	LabelXIDCode = "xid_code"
)
