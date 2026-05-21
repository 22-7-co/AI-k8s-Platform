package healing

import (
	"context"
	"fmt"

	"github.com/ai-k8s-platform/core/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StepAction describes what AdvanceHealing executed or would execute (dry-run).
type StepAction string

const (
	ActionNone   StepAction = "none"
	ActionCordon StepAction = "cordon"
	ActionTaint  StepAction = "taint"
)

// AdvanceHealing reads healing-state on the node and performs exactly one workflow step.
// P0 covers "" -> cordoned -> tainted; eviction is deferred to P2.
func AdvanceHealing(ctx context.Context, client kubernetes.Interface, nodeName string, dryRun bool) (StepAction, string, error) {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return ActionNone, "", fmt.Errorf("get node %q: %w", nodeName, err)
	}

	state := GetHealingState(node)
	switch state {
	case "", labels.StateCordoned:
		if state == labels.StateCordoned {
			return advanceTaint(ctx, client, nodeName, dryRun)
		}
		return advanceCordon(ctx, client, nodeName, dryRun)
	case labels.StateTainted, labels.StateEvicted, labels.StateCompleted:
		return ActionNone, state, nil
	default:
		return ActionNone, state, fmt.Errorf("unknown healing-state %q on node %q", state, nodeName)
	}
}

func advanceCordon(ctx context.Context, client kubernetes.Interface, nodeName string, dryRun bool) (StepAction, string, error) {
	if dryRun {
		return ActionCordon, labels.StateCordoned, nil
	}
	if err := Cordon(ctx, client, nodeName); err != nil {
		return ActionNone, "", err
	}
	if err := SetHealingState(ctx, client, nodeName, labels.StateCordoned); err != nil {
		return ActionNone, "", err
	}
	return ActionCordon, labels.StateCordoned, nil
}

func advanceTaint(ctx context.Context, client kubernetes.Interface, nodeName string, dryRun bool) (StepAction, string, error) {
	if dryRun {
		return ActionTaint, labels.StateTainted, nil
	}
	if err := AddGPUTaint(ctx, client, nodeName); err != nil {
		return ActionNone, "", err
	}
	if err := SetHealingState(ctx, client, nodeName, labels.StateTainted); err != nil {
		return ActionNone, "", err
	}
	return ActionTaint, labels.StateTainted, nil
}
