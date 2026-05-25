// Package labels defines Kubernetes label, annotation, and taint constants for the platform.
package labels

const (
	// LabelHealingState is the node label key for healing workflow checkpoint.
	LabelHealingState = "ai-k8s-platform.io/healing-state"

	// Healing state values (stored in LabelHealingState).
	StateCordoned  = "cordoned"
	StateTainted   = "tainted"
	StateEvicted   = "evicted"
	StateCompleted = "completed"

	// TaintKeyGPUFault is applied when a GPU hardware fault is detected.
	TaintKeyGPUFault = "ai-k8s-platform.io/gpu-fault"

	// TaintEffect is the effect used for GPU fault taints.
	TaintEffect = "NoSchedule"

	// LabelTrainingWorkload marks training pods subject to eviction.
	LabelTrainingWorkload = "ai-k8s-platform.io/training"

	// TrainingWorkloadValue is the expected value for LabelTrainingWorkload.
	TrainingWorkloadValue = "true"

	// AnnotationHealingCompletedAt records when healing completed (RFC3339).
	AnnotationHealingCompletedAt = "ai-k8s-platform.io/healing-completed-at"

	// AnnotationHealingFailCount counts consecutive handle-node failures.
	AnnotationHealingFailCount = "ai-k8s-platform.io/healing-fail-count"

	// DefaultTrainingSelector is the label selector for training pods.
	DefaultTrainingSelector = LabelTrainingWorkload + "=" + TrainingWorkloadValue
)
