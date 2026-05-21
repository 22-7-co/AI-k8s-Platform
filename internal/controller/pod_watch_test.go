package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func jobPod(name, ns, node string, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    map[string]string{"batch.kubernetes.io/job-name": "training-job"},
		},
		Spec:   corev1.PodSpec{NodeName: node},
		Status: corev1.PodStatus{Phase: phase},
	}
}

func TestWaitForReschedule(t *testing.T) {
	t.Parallel()
	old := jobPod("old", "ai-training", "node-1", corev1.PodRunning)
	newPod := jobPod("new", "ai-training", "node-2", corev1.PodRunning)
	client := fake.NewSimpleClientset(old, newPod)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := WaitForReschedule(ctx, client, "training-job", "ai-training", "node-1", 3*time.Second); err != nil {
		t.Fatalf("WaitForReschedule: %v", err)
	}
}
