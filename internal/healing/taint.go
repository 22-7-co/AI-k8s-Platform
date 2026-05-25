package healing

import (
	"context"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// AddGPUTaint adds the GPU fault NoSchedule taint if not already present.
func AddGPUTaint(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	return updateNodeWithRetry(ctx, client, nodeName, func(node *corev1.Node) bool {
		if gpuTaintPresent(node) {
			return false
		}
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Key:    labels.TaintKeyGPUFault,
			Effect: corev1.TaintEffectNoSchedule,
		})
		return true
	})
}

func gpuTaintPresent(node *corev1.Node) bool {
	for _, t := range node.Spec.Taints {
		if t.Key == labels.TaintKeyGPUFault && t.Effect == corev1.TaintEffectNoSchedule {
			return true
		}
	}
	return false
}
