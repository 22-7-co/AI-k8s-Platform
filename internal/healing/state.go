package healing

import (
	"context"
	"fmt"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SetHealingState writes the healing-state label on a node.
func SetHealingState(ctx context.Context, client kubernetes.Interface, nodeName, state string) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node %q: %w", nodeName, err)
	}
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	node.Labels[labels.LabelHealingState] = state
	if _, err := client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update node %q healing-state: %w", nodeName, err)
	}
	return nil
}

// GetHealingState returns the current healing-state label value (empty if unset).
func GetHealingState(node *corev1.Node) string {
	if node == nil || node.Labels == nil {
		return ""
	}
	return node.Labels[labels.LabelHealingState]
}
