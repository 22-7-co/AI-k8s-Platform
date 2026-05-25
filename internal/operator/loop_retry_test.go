package operator

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/ai-k8s-platform/core/internal/operator/config"
	"github.com/ai-k8s-platform/core/internal/prometheus"
	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestHandleNodeWithRetry_maxExceeded_incrementsFailCount(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := fake.NewSimpleClientset(node)
	failCount := 0
	client.PrependReactor("update", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		u := action.(ktesting.UpdateAction)
		n := u.GetObject().(*corev1.Node)
		newCount := 0
		if n.Annotations != nil {
			if v := n.Annotations[labels.AnnotationHealingFailCount]; v != "" {
				newCount, _ = strconv.Atoi(v)
			}
		}
		if newCount > failCount {
			failCount = newCount
			return false, nil, nil
		}
		return true, nil, errors.New("simulated update failure")
	})

	cfg := config.Load()
	cfg.HealingDryRun = false
	cfg.HealingMaxRetries = 1
	cfg.RetryBackoffBase = time.Millisecond
	cfg.PrometheusMockNodes = []string{"node-1"}

	loop := &Loop{
		K8s:    client,
		Prom:   prometheus.NewMockClient([]string{"node-1"}),
		Config: cfg,
	}
	loop.handleNodeWithRetry(context.Background(), "node-1")

	got, err := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if got.Annotations[labels.AnnotationHealingFailCount] != "2" {
		t.Fatalf("fail-count=%q, want 2 (initial attempt + one retry)", got.Annotations[labels.AnnotationHealingFailCount])
	}
}
