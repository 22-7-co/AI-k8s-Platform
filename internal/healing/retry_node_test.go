package healing

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestCordon_Conflict_retry(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1", ResourceVersion: "1"},
		Spec:       corev1.NodeSpec{Unschedulable: false},
	}
	client := fake.NewSimpleClientset(node)
	updates := 0
	client.PrependReactor("update", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		updates++
		if updates == 1 {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "nodes"}, "node-1", nil)
		}
		return false, nil, nil
	})

	if err := Cordon(context.Background(), client, "node-1"); err != nil {
		t.Fatalf("Cordon: %v", err)
	}
	if updates < 2 {
		t.Fatalf("updates = %d, want >= 2", updates)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if !got.Spec.Unschedulable {
		t.Fatal("expected cordoned")
	}
}

func TestAddGPUTaint_Conflict_retry(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}, Spec: corev1.NodeSpec{}}
	client := fake.NewSimpleClientset(node)
	updates := 0
	client.PrependReactor("update", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		updates++
		if updates == 1 {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "nodes"}, "node-1", nil)
		}
		return false, nil, nil
	})

	if err := AddGPUTaint(context.Background(), client, "node-1"); err != nil {
		t.Fatalf("AddGPUTaint: %v", err)
	}
	got, _ := client.CoreV1().Nodes().Get(context.Background(), "node-1", metav1.GetOptions{})
	if !hasGPUTaint(got) {
		t.Fatal("expected taint")
	}
}
