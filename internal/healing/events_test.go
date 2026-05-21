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

func TestRecordHealingEvent(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1", UID: "uid-1"},
	}
	client := fake.NewSimpleClientset(node)
	var created int
	client.PrependReactor("create", "events", func(action ktesting.Action) (bool, runtime.Object, error) {
		created++
		ev := action.(ktesting.CreateAction).GetObject().(*corev1.Event)
		if ev.InvolvedObject.Name != "node-1" {
			t.Fatalf("involved = %s", ev.InvolvedObject.Name)
		}
		if ev.Reason != "HealingCordon" {
			t.Fatalf("reason = %s", ev.Reason)
		}
		return true, ev, nil
	})

	opts := Options{DryRun: false}
	if _, _, err := AdvanceHealing(context.Background(), client, "node-1", opts); err != nil {
		t.Fatalf("AdvanceHealing: %v", err)
	}
	if created == 0 {
		t.Fatal("expected healing event")
	}
}
