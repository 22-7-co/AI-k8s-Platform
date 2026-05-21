package operator

import (
	"context"
	"testing"
	"time"

	"github.com/ai-k8s-platform/core/internal/operator/config"
	"github.com/ai-k8s-platform/core/internal/prometheus"
	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHandleNode_cooldown_skip(t *testing.T) {
	t.Parallel()
	completed := &corev1.Node{
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
	client := fake.NewSimpleClientset(completed)
	cfg := config.Load()
	cfg.PrometheusMockNodes = []string{"node-1"}
	loop := &Loop{K8s: client, Prom: prometheus.NewMockClient([]string{"node-1"}), Config: cfg}
	if err := loop.handleNode(context.Background(), "node-1"); err != nil {
		t.Fatalf("handleNode: %v", err)
	}
}

func TestLoop_tick_dryRun(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := fake.NewSimpleClientset(node)
	cfg := config.Load()
	cfg.PrometheusMock = true
	cfg.PrometheusMockNodes = []string{"node-1"}
	cfg.HealingDryRun = true
	cfg.PollInterval = time.Hour

	loop := &Loop{
		K8s:    client,
		Prom:   prometheus.NewMockClient(cfg.PrometheusMockNodes),
		Config: cfg,
	}
	if err := loop.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if got.Spec.Unschedulable {
		t.Fatal("dry-run should not cordon")
	}
}
