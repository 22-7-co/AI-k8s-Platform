package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// WaitForReschedule waits until a Job pod is Running on a healthy node.
// If the cluster has only one node, excludeNode is ignored so the Job may reschedule there after uncordon/untaint.
func WaitForReschedule(ctx context.Context, client kubernetes.Interface, jobName, namespace, excludeNode string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	selector := labels.Set{"batch.kubernetes.io/job-name": jobName}.AsSelector().String()
	if singleNodeCluster(ctx, client) {
		excludeNode = ""
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return fmt.Errorf("list job pods: %w", err)
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Spec.NodeName == "" || pod.Spec.NodeName == excludeNode {
				continue
			}
			if pod.Status.Phase == corev1.PodRunning {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	if excludeNode != "" {
		return fmt.Errorf("timeout waiting for job %s/%s pod Running outside node %s", namespace, jobName, excludeNode)
	}
	return fmt.Errorf("timeout waiting for job %s/%s pod Running", namespace, jobName)
}

func singleNodeCluster(ctx context.Context, client kubernetes.Interface) bool {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false
	}
	return len(nodes.Items) < 2
}
