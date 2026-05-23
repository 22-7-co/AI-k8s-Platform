package healing

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Cordon marks a node unschedulable. It is a no-op if already cordoned.
func Cordon(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	return updateNodeWithRetry(ctx, client, nodeName, func(node *corev1.Node) bool {
		if node.Spec.Unschedulable {
			return false
		}
		node.Spec.Unschedulable = true
		return true
	})
}
