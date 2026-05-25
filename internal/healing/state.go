package healing

import (
	"context"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// SetHealingState writes the healing-state label on a node.
func SetHealingState(ctx context.Context, client kubernetes.Interface, nodeName, state string) error {
	return updateNodeWithRetry(ctx, client, nodeName, func(node *corev1.Node) bool {
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		if node.Labels[labels.LabelHealingState] == state {
			return false
		}
		node.Labels[labels.LabelHealingState] = state
		return true
	})
}

// GetHealingState returns the current healing-state label value (empty if unset).
func GetHealingState(node *corev1.Node) string {
	if node == nil || node.Labels == nil {
		return ""
	}
	return node.Labels[labels.LabelHealingState]
}
