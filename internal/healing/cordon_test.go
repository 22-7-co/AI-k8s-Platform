package healing

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestCordon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		node       *corev1.Node
		nodeName   string
		wantUnsched bool
		wantErr    bool
		maxUpdates int
	}{
		{
			name: "cordon_sets_unschedulable",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
				Spec:       corev1.NodeSpec{Unschedulable: false},
			},
			nodeName:    "node-1",
			wantUnsched: true,
			maxUpdates:  1,
		},
		{
			name: "cordon_idempotent",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
				Spec:       corev1.NodeSpec{Unschedulable: true},
			},
			nodeName:    "node-2",
			wantUnsched: true,
			maxUpdates:  0,
		},
		{
			name:     "cordon_not_found",
			node:     nil,
			nodeName: "missing",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var objects []runtime.Object
			if tt.node != nil {
				objects = append(objects, tt.node)
			}
			client := fake.NewSimpleClientset(objects...)
			updates := 0
			client.PrependReactor("update", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
				updates++
				return false, nil, nil
			})

			err := Cordon(context.Background(), client, tt.nodeName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Cordon: %v", err)
			}
			if updates > tt.maxUpdates {
				t.Fatalf("Update calls = %d, want <= %d", updates, tt.maxUpdates)
			}
			got, err := client.CoreV1().Nodes().Get(context.Background(), tt.nodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Get node: %v", err)
			}
			if got.Spec.Unschedulable != tt.wantUnsched {
				t.Fatalf("Unschedulable = %v, want %v", got.Spec.Unschedulable, tt.wantUnsched)
			}
		})
	}
}
