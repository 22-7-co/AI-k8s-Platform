package healing

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RecordHealingEvent writes an audit event on the node (involved object).
func RecordHealingEvent(ctx context.Context, client kubernetes.Interface, nodeName, action, message string) error {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get node for event: %w", err)
	}

	ref := &corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Node",
		Name:       node.Name,
		UID:        node.UID,
	}

	now := metav1.Now()
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%x", nodeName, now.UnixNano()),
			Namespace: metav1.NamespaceDefault,
		},
		InvolvedObject: *ref,
		Reason:         healingReason(action),
		Message:        message,
		Type:           corev1.EventTypeNormal,
		FirstTimestamp: now,
		LastTimestamp:  now,
		Count:          1,
		Source: corev1.EventSource{
			Component: "ai-k8s-platform/operator",
		},
	}

	_, err = client.CoreV1().Events(metav1.NamespaceDefault).Create(ctx, event, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create healing event: %w", err)
	}
	return nil
}

func healingReason(action string) string {
	if action == "" {
		return "Healing"
	}
	return "Healing" + strings.ToUpper(action[:1]) + action[1:]
}
