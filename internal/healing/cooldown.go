package healing

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// InCooldown reports whether the node completed healing recently within cooldown.
func InCooldown(node *corev1.Node, cooldown time.Duration) bool {
	if node == nil || GetHealingState(node) != labels.StateCompleted {
		return false
	}
	if node.Annotations == nil {
		return true
	}
	at, ok := node.Annotations[labels.AnnotationHealingCompletedAt]
	if !ok || at == "" {
		return true
	}
	ts, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return true
	}
	return time.Since(ts) < cooldown
}

// MarkCompleted sets healing-state=completed and records completion time.
func MarkCompleted(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node %q: %w", nodeName, err)
	}
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	if node.Annotations == nil {
		node.Annotations = make(map[string]string)
	}
	node.Labels[labels.LabelHealingState] = labels.StateCompleted
	node.Annotations[labels.AnnotationHealingCompletedAt] = time.Now().UTC().Format(time.RFC3339)
	if _, err := client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("mark completed %q: %w", nodeName, err)
	}
	return nil
}
