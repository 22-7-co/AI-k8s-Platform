package healing

import (
	"context"
	"testing"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAddGPUTaint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		node      *corev1.Node
		wantTaint bool
		wantCount int
	}{
		{
			name: "taint_adds_gpu_fault",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
				Spec:       corev1.NodeSpec{},
			},
			wantTaint: true,
			wantCount: 1,
		},
		{
			name: "taint_idempotent",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    labels.TaintKeyGPUFault,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			wantTaint: true,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := fake.NewSimpleClientset(tt.node)
			if err := AddGPUTaint(context.Background(), client, tt.node.Name); err != nil {
				t.Fatalf("AddGPUTaint: %v", err)
			}
			got, err := client.CoreV1().Nodes().Get(context.Background(), tt.node.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if hasGPUTaint(got) != tt.wantTaint {
				t.Fatalf("hasGPUTaint = %v, want %v", hasGPUTaint(got), tt.wantTaint)
			}
			if len(got.Spec.Taints) != tt.wantCount {
				t.Fatalf("taints len = %d, want %d", len(got.Spec.Taints), tt.wantCount)
			}
			if tt.name == "taint_idempotent" {
				if err := AddGPUTaint(context.Background(), client, tt.node.Name); err != nil {
					t.Fatalf("second AddGPUTaint: %v", err)
				}
				got2, _ := client.CoreV1().Nodes().Get(context.Background(), tt.node.Name, metav1.GetOptions{})
				if len(got2.Spec.Taints) != 1 {
					t.Fatalf("idempotent: taints len = %d, want 1", len(got2.Spec.Taints))
				}
			}
		})
	}
}

func TestSetHealingState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		node      *corev1.Node
		newState  string
		wantState string
	}{
		{
			name: "set_state_cordoned",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			},
			newState:  labels.StateCordoned,
			wantState: labels.StateCordoned,
		},
		{
			name: "set_state_overwrite",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-2",
					Labels: map[string]string{
						labels.LabelHealingState: labels.StateCordoned,
					},
				},
			},
			newState:  labels.StateTainted,
			wantState: labels.StateTainted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := fake.NewSimpleClientset(tt.node)
			if err := SetHealingState(context.Background(), client, tt.node.Name, tt.newState); err != nil {
				t.Fatalf("SetHealingState: %v", err)
			}
			got, err := client.CoreV1().Nodes().Get(context.Background(), tt.node.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.Labels[labels.LabelHealingState] != tt.wantState {
				t.Fatalf("healing-state = %q, want %q", got.Labels[labels.LabelHealingState], tt.wantState)
			}
		})
	}
}

func hasGPUTaint(node *corev1.Node) bool {
	for _, t := range node.Spec.Taints {
		if t.Key == labels.TaintKeyGPUFault && t.Effect == corev1.TaintEffectNoSchedule {
			return true
		}
	}
	return false
}
