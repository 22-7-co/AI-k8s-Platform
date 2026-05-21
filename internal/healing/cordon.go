package healing

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Cordon marks a node unschedulable. It is a no-op if already cordoned.
func Cordon(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node %q: %w", nodeName, err)
	}
	if node.Spec.Unschedulable {
		return nil
	}
	node.Spec.Unschedulable = true
	if _, err := client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update node %q unschedulable: %w", nodeName, err)
	}
	return nil
}
