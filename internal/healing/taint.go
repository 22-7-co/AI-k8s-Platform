package healing

import (
	"context"
	"fmt"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// AddGPUTaint adds the GPU fault NoSchedule taint if not already present.
func AddGPUTaint(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node %q: %w", nodeName, err)
	}
	if gpuTaintPresent(node) {
		return nil
	}
	node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
		Key:    labels.TaintKeyGPUFault,
		Effect: corev1.TaintEffectNoSchedule,
	})
	if _, err := client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update node %q taints: %w", nodeName, err)
	}
	return nil
}

func gpuTaintPresent(node *corev1.Node) bool {
	for _, t := range node.Spec.Taints {
		if t.Key == labels.TaintKeyGPUFault && t.Effect == corev1.TaintEffectNoSchedule {
			return true
		}
	}
	return false
}
