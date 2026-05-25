package healing

import (
	"github.com/ai-k8s-platform/core/pkg/labels"
)

// Options configures healing steps (namespaces, selector, dry-run).
type Options struct {
	DryRun             bool
	TargetNamespaces   []string
	TrainingSelector   string
	SkipEvents         bool
}

// DefaultOptions returns P2 defaults aligned with the project plan.
func DefaultOptions() Options {
	return Options{
		DryRun:           true,
		TargetNamespaces: []string{"ai-training"},
		TrainingSelector: labels.DefaultTrainingSelector,
	}
}
