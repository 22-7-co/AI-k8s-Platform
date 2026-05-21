package healing

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// ListTrainingPodsOnNode returns training pods scheduled on nodeName in allowed namespaces.
func ListTrainingPodsOnNode(ctx context.Context, client kubernetes.Interface, nodeName, selector string, namespaces []string) ([]*corev1.Pod, error) {
	sel, err := k8slabels.Parse(selector)
	if err != nil {
		return nil, fmt.Errorf("parse selector %q: %w", selector, err)
	}

	nsSet := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		nsSet[ns] = struct{}{}
	}
	searchAll := len(nsSet) == 0

	var out []*corev1.Pod
	podList, err := client.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return nil, fmt.Errorf("list pods on node %q: %w", nodeName, err)
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Spec.NodeName != nodeName {
			continue
		}
		if !searchAll {
			if _, ok := nsSet[pod.Namespace]; !ok {
				continue
			}
		}
		if !sel.Matches(k8slabels.Set(pod.Labels)) {
			continue
		}
		if isDaemonSetPod(pod) {
			continue
		}
		out = append(out, pod)
	}
	return out, nil
}

func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// EvictPods evicts pods via policy/v1 Eviction, falling back to delete on failure.
func EvictPods(ctx context.Context, client kubernetes.Interface, pods []*corev1.Pod) (int, error) {
	var evicted int
	for _, pod := range pods {
		if pod == nil {
			continue
		}
		if err := evictOnePod(ctx, client, pod); err != nil {
			return evicted, err
		}
		evicted++
	}
	return evicted, nil
}

func evictOnePod(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod) error {
	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &metav1.DeleteOptions{},
	}
	err := client.PolicyV1().Evictions(pod.Namespace).Evict(ctx, eviction)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		if !apierrors.IsTooManyRequests(err) && !apierrors.IsForbidden(err) && !apierrors.IsTimeout(err) {
			return fmt.Errorf("evict %s/%s: %w", pod.Namespace, pod.Name, err)
		}
	}
	if err := deletePodIfPresent(ctx, client, pod.Namespace, pod.Name); err != nil {
		return err
	}
	return waitPodGone(ctx, client, pod.Namespace, pod.Name)
}

func deletePodIfPresent(ctx context.Context, client kubernetes.Interface, ns, name string) error {
	_, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	delErr := client.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if delErr != nil && !apierrors.IsNotFound(delErr) {
		return delErr
	}
	return nil
}

func waitPodGone(ctx context.Context, client kubernetes.Interface, ns, name string) error {
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		pod, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if pod.DeletionTimestamp != nil {
			return nil
		}
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("pod %s/%s still exists after eviction", ns, name)
}
