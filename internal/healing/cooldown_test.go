package healing

import (
	"context"
	"testing"
	"time"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInCooldown_completed_within_cooldown(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
			Labels: map[string]string{
				labels.LabelHealingState: labels.StateCompleted,
			},
			Annotations: map[string]string{
				labels.AnnotationHealingCompletedAt: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if !InCooldown(node, 10*time.Minute) {
		t.Fatal("expected in cooldown")
	}
}

func TestInCooldown_completed_after_cooldown(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
			Labels: map[string]string{
				labels.LabelHealingState: labels.StateCompleted,
			},
			Annotations: map[string]string{
				labels.AnnotationHealingCompletedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			},
		},
	}
	if InCooldown(node, 10*time.Minute) {
		t.Fatal("expected not in cooldown")
	}
}

func TestIncrementFailCount_Retry(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := fake.NewSimpleClientset(node)
	if err := IncrementFailCount(context.Background(), client, "node-1"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := IncrementFailCount(context.Background(), client, "node-1"); err != nil {
		t.Fatalf("second: %v", err)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if got.Annotations[labels.AnnotationHealingFailCount] != "2" {
		t.Fatalf("count=%q", got.Annotations[labels.AnnotationHealingFailCount])
	}
}

func TestMarkCompleted(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := fake.NewSimpleClientset(node)
	if err := MarkCompleted(context.Background(), client, "node-1"); err != nil {
		t.Fatalf("MarkCompleted: %v", err)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if got.Labels[labels.LabelHealingState] != labels.StateCompleted {
		t.Fatalf("state=%q", got.Labels[labels.LabelHealingState])
	}
	if got.Annotations[labels.AnnotationHealingCompletedAt] == "" {
		t.Fatal("missing completed-at annotation")
	}
}
