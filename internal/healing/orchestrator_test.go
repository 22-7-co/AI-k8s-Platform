package healing

import (
	"context"
	"testing"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAdvanceHealing_empty_to_cordoned(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec:       corev1.NodeSpec{Unschedulable: false},
	}
	client := fake.NewSimpleClientset(node)

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", false)
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
	if got.Labels[labels.LabelHealingState] != labels.StateCordoned {
		t.Fatalf("healing-state=%q", got.Labels[labels.LabelHealingState])
	}
}

func TestAdvanceHealing_cordoned_to_tainted(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
			Labels: map[string]string{
				labels.LabelHealingState: labels.StateCordoned,
			},
		},
		Spec: corev1.NodeSpec{Unschedulable: true},
	}
	client := fake.NewSimpleClientset(node)

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", false)
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionTaint || state != labels.StateTainted {
		t.Fatalf("action=%q state=%q, want taint/tainted", action, state)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if !hasGPUTaint(got) {
		t.Fatal("expected GPU fault taint")
	}
	if got.Labels[labels.LabelHealingState] != labels.StateTainted {
		t.Fatalf("healing-state=%q", got.Labels[labels.LabelHealingState])
	}
}

func TestAdvanceHealing_full_chain_idempotent_skip(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec:       corev1.NodeSpec{Unschedulable: false},
	}
	client := fake.NewSimpleClientset(node)

	if _, _, err := AdvanceHealing(context.Background(), client, "node-1", false); err != nil {
		t.Fatalf("step1: %v", err)
	}
	if _, _, err := AdvanceHealing(context.Background(), client, "node-1", false); err != nil {
		t.Fatalf("step2: %v", err)
	}
	action, state, err := AdvanceHealing(context.Background(), client, "node-1", false)
	if err != nil {
		t.Fatalf("step3 tainted noop: %v", err)
	}
	if action != ActionNone || state != labels.StateTainted {
		t.Fatalf("action=%q state=%q, want none/tainted", action, state)
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
			action, state, err := AdvanceHealing(context.Background(), client, "node-x", false)
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
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec:       corev1.NodeSpec{Unschedulable: false},
	}
	client := fake.NewSimpleClientset(node)

	action, state, err := AdvanceHealing(context.Background(), client, "node-1", true)
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
	if _, ok := got.Labels[labels.LabelHealingState]; ok {
		t.Fatal("dry-run must not set healing-state")
	}
}

func TestAdvanceHealing_cordoned_skips_second_cordon(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
			Labels: map[string]string{
				labels.LabelHealingState: labels.StateCordoned,
			},
		},
		Spec: corev1.NodeSpec{Unschedulable: true},
	}
	client := fake.NewSimpleClientset(node)

	action, _, err := AdvanceHealing(context.Background(), client, "node-1", false)
	if err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if action != ActionTaint {
		t.Fatalf("action=%q, want taint only", action)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if !got.Spec.Unschedulable {
		t.Fatal("Unschedulable should remain true")
	}
}
