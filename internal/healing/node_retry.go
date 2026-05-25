package healing

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const maxNodeUpdateRetries = 3

// updateNodeWithRetry applies mutate to a node and persists with conflict retry.
// mutate returns true if the node was changed and needs an Update.
func updateNodeWithRetry(ctx context.Context, client kubernetes.Interface, nodeName string, mutate func(*corev1.Node) bool) error {
	var lastErr error
	for attempt := 0; attempt < maxNodeUpdateRetries; attempt++ {
		node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get node %q: %w", nodeName, err)
		}
		if !mutate(node) {
			return nil
		}
		_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}
		lastErr = err
		if !isUpdateConflict(err) {
			return fmt.Errorf("update node %q: %w", nodeName, err)
		}
	}
	return fmt.Errorf("update node %q after %d retries: %w", nodeName, maxNodeUpdateRetries, lastErr)
}

func isUpdateConflict(err error) bool {
	if err == nil {
		return false
	}
	if apierrors.IsConflict(err) {
		return true
	}
	return strings.Contains(err.Error(), "the object has been modified")
}
