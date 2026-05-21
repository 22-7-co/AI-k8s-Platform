package healing

import (
	"context"
	"testing"

	"github.com/ai-k8s-platform/core/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func trainingPod(name, ns, node string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				labels.LabelTrainingWorkload: labels.TrainingWorkloadValue,
			},
		},
		Spec: corev1.PodSpec{NodeName: node},
	}
}

func TestListTrainingPodsOnNode(t *testing.T) {
	t.Parallel()
	pods := []runtime.Object{
		trainingPod("train-1", "ai-training", "node-1"),
		trainingPod("other", "default", "node-1"),
		trainingPod("train-2", "ai-training", "node-2"),
	}
	client := fake.NewSimpleClientset(pods...)
	got, err := ListTrainingPodsOnNode(context.Background(), client, "node-1", labels.DefaultTrainingSelector, []string{"ai-training"})
	if err != nil {
		t.Fatalf("ListTrainingPodsOnNode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "train-1" {
		t.Fatalf("pods = %v, want [train-1]", names(got))
	}
}

func TestEvictPods_uses_eviction_api(t *testing.T) {
	t.Parallel()
	pod := trainingPod("train-1", "ai-training", "node-1")
	client := fake.NewSimpleClientset(pod)
	evictions := 0
	client.PrependReactor("create", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		evictions++
		ev := action.(ktesting.CreateAction).GetObject().(*policyv1.Eviction)
		if ev.Name != "train-1" {
			t.Fatalf("eviction name = %s", ev.Name)
		}
		return true, nil, apierrors.NewTooManyRequests("rate", 1)
	})

	n, err := EvictPods(context.Background(), client, []*corev1.Pod{pod})
	if err != nil {
		t.Fatalf("EvictPods: %v", err)
	}
	if n != 1 || evictions != 1 {
		t.Fatalf("evicted=%d evictions=%d", n, evictions)
	}
}

func TestEvictPods_fallback_delete(t *testing.T) {
	t.Parallel()
	pod := trainingPod("train-1", "ai-training", "node-1")
	client := fake.NewSimpleClientset(pod)
	client.PrependReactor("create", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		return true, nil, apierrors.NewTooManyRequests("rate", 1)
	})

	n, err := EvictPods(context.Background(), client, []*corev1.Pod{pod})
	if err != nil {
		t.Fatalf("EvictPods: %v", err)
	}
	if n != 1 {
		t.Fatalf("evicted = %d", n)
	}
	_, getErr := client.CoreV1().Pods("ai-training").Get(context.Background(), "train-1", metav1.GetOptions{})
	if !apierrors.IsNotFound(getErr) {
		t.Fatalf("pod should be deleted: %v", getErr)
	}
}

func TestEvictPods_skips_daemonset(t *testing.T) {
	t.Parallel()
	dsPod := trainingPod("ds-1", "ai-training", "node-1")
	dsPod.OwnerReferences = []metav1.OwnerReference{{Kind: "DaemonSet", Name: "ds", Controller: ptr(true)}}
	client := fake.NewSimpleClientset(dsPod, trainingPod("train-1", "ai-training", "node-1"))
	got, err := ListTrainingPodsOnNode(context.Background(), client, "node-1", labels.DefaultTrainingSelector, []string{"ai-training"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Name != "train-1" {
		t.Fatalf("got %v", names(got))
	}
}

func TestEvictPods_none_ok(t *testing.T) {
	t.Parallel()
	client := fake.NewSimpleClientset()
	n, err := EvictPods(context.Background(), client, nil)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func names(pods []*corev1.Pod) []string {
	out := make([]string, len(pods))
	for i, p := range pods {
		out[i] = p.Name
	}
	return out
}

func ptr(b bool) *bool { return &b }
