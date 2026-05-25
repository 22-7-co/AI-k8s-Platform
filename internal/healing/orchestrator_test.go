package healing

import (
	"context"
	"testing"

	"github.com/ai-k8s-platform/core/pkg/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func testOpts() Options {
	return Options{
		DryRun:             false,
		TargetNamespaces:   []string{"ai-training"},
		TrainingSelector:   labels.DefaultTrainingSelector,
		SkipEvents:         true,
	}
}

func TestAdvanceHealing_empty_to_cordoned(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec:       corev1.NodeSpec{Unschedulable: false},
	}
	client := fake.NewSimpleClientset(node)

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", testOpts())
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionCordon || state != labels.StateCordoned {
		t.Fatalf("action=%q state=%q, want cordon/cordoned", action, state)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if !got.Spec.Unschedulable {
		t.Fatal("expected Unschedulable=true")
	}
}

func TestAdvanceHealing_cordoned_to_tainted(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-1",
			Labels: map[string]string{labels.LabelHealingState: labels.StateCordoned},
		},
		Spec: corev1.NodeSpec{Unschedulable: true},
	}
	client := fake.NewSimpleClientset(node)

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", testOpts())
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionTaint || state != labels.StateTainted {
		t.Fatalf("action=%q state=%q", action, state)
	}
}

func TestAdvanceHealing_tainted_to_evicted(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-1",
			Labels: map[string]string{labels.LabelHealingState: labels.StateTainted},
		},
	}
	pod := trainingPod("train-1", "ai-training", "node-1")
	client := fake.NewSimpleClientset(node, pod)
	client.PrependReactor("create", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		return true, nil, apierrors.NewTooManyRequests("rate", 1)
	})

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", testOpts())
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionEvict || state != labels.StateEvicted {
		t.Fatalf("action=%q state=%q", action, state)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if got.Labels[labels.LabelHealingState] != labels.StateEvicted {
		t.Fatalf("state=%q", got.Labels[labels.LabelHealingState])
	}
}

func TestAdvanceHealing_full_chain_to_evicted(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	pod := trainingPod("train-1", "ai-training", "node-1")
	client := fake.NewSimpleClientset(node, pod)
	client.PrependReactor("create", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		return true, nil, apierrors.NewTooManyRequests("rate", 1)
	})
	opts := testOpts()

	for i := 0; i < 3; i++ {
		action, _, err := AdvanceHealing(context.Background(), client, "node-1", opts)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if i < 2 && action == ActionNone {
			t.Fatalf("step %d unexpected none", i)
		}
	}
	action, state, err := AdvanceHealing(context.Background(), client, "node-1", opts)
	if err != nil || action != ActionNone || state != labels.StateEvicted {
		t.Fatalf("final action=%q state=%q err=%v", action, state, err)
	}
}

func TestAdvanceHealing_terminal_states_noop(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		state string
	}{
		{"evicted", labels.StateEvicted},
		{"completed", labels.StateCompleted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "node-x",
					Labels: map[string]string{labels.LabelHealingState: tc.state},
				},
			}
			client := fake.NewSimpleClientset(node)
			action, state, err := AdvanceHealing(context.Background(), client, "node-x", testOpts())
			if err != nil {
				t.Fatalf("AdvanceHealing: %v", err)
			}
			if action != ActionNone || state != tc.state {
				t.Fatalf("action=%q state=%q", action, state)
			}
		})
	}
}

func TestAdvanceHealing_dryRun_no_mutation(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := fake.NewSimpleClientset(node)
	opts := testOpts()
	opts.DryRun = true

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", opts)
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionCordon || state != labels.StateCordoned {
		t.Fatalf("action=%q state=%q", action, state)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if got.Spec.Unschedulable {
		t.Fatal("dry-run must not cordon")
	}
}

func TestAdvanceHealing_cordoned_skips_second_cordon(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-1",
			Labels: map[string]string{labels.LabelHealingState: labels.StateCordoned},
		},
		Spec: corev1.NodeSpec{Unschedulable: true},
	}
	client := fake.NewSimpleClientset(node)

	action, _, err := AdvanceHealing(context.Background(), client, "node-1", testOpts())
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionTaint {
		t.Fatalf("action=%q, want taint only", action)
	}
}
