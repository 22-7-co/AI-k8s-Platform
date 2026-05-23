package healing

import (
	"context"
	"strconv"
	"time"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
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
	now := time.Now().UTC().Format(time.RFC3339)
	return updateNodeWithRetry(ctx, client, nodeName, func(node *corev1.Node) bool {
		changed := false
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		if node.Labels[labels.LabelHealingState] != labels.StateCompleted {
			node.Labels[labels.LabelHealingState] = labels.StateCompleted
			changed = true
		}
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}
		if node.Annotations[labels.AnnotationHealingCompletedAt] != now {
			node.Annotations[labels.AnnotationHealingCompletedAt] = now
			changed = true
		}
		return changed
	})
}

// IncrementFailCount bumps healing-fail-count on the node annotation.
func IncrementFailCount(ctx context.Context, client kubernetes.Interface, nodeName string) error {
	return updateNodeWithRetry(ctx, client, nodeName, func(node *corev1.Node) bool {
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}
		prev := 0
		if v := node.Annotations[labels.AnnotationHealingFailCount]; v != "" {
			prev, _ = strconv.Atoi(v)
		}
		node.Annotations[labels.AnnotationHealingFailCount] = strconv.Itoa(prev + 1)
		return true
	})
}
