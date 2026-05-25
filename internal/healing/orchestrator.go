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
	ActionEvict  StepAction = "evict"
)

// AdvanceHealing reads healing-state on the node and performs exactly one workflow step.
func AdvanceHealing(ctx context.Context, client kubernetes.Interface, nodeName string, opts Options) (StepAction, string, error) {
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return ActionNone, "", fmt.Errorf("get node %q: %w", nodeName, err)
	}

	state := GetHealingState(node)
	switch state {
	case "":
		return advanceCordon(ctx, client, nodeName, opts)
	case labels.StateCordoned:
		return advanceTaint(ctx, client, nodeName, opts)
	case labels.StateTainted:
		return advanceEvict(ctx, client, nodeName, opts)
	case labels.StateEvicted, labels.StateCompleted:
		return ActionNone, state, nil
	default:
		return ActionNone, state, fmt.Errorf("unknown healing-state %q on node %q", state, nodeName)
	}
}

func advanceCordon(ctx context.Context, client kubernetes.Interface, nodeName string, opts Options) (StepAction, string, error) {
	if opts.DryRun {
		return ActionCordon, labels.StateCordoned, nil
	}
	if err := Cordon(ctx, client, nodeName); err != nil {
		return ActionNone, "", err
	}
	if err := SetHealingState(ctx, client, nodeName, labels.StateCordoned); err != nil {
		return ActionNone, "", err
	}
	recordEvent(ctx, client, nodeName, opts, "cordon", "node cordoned")
	return ActionCordon, labels.StateCordoned, nil
}

func advanceTaint(ctx context.Context, client kubernetes.Interface, nodeName string, opts Options) (StepAction, string, error) {
	if opts.DryRun {
		return ActionTaint, labels.StateTainted, nil
	}
	if err := AddGPUTaint(ctx, client, nodeName); err != nil {
		return ActionNone, "", err
	}
	if err := SetHealingState(ctx, client, nodeName, labels.StateTainted); err != nil {
		return ActionNone, "", err
	}
	recordEvent(ctx, client, nodeName, opts, "taint", "gpu fault taint applied")
	return ActionTaint, labels.StateTainted, nil
}

func advanceEvict(ctx context.Context, client kubernetes.Interface, nodeName string, opts Options) (StepAction, string, error) {
	if opts.DryRun {
		return ActionEvict, labels.StateEvicted, nil
	}
	selector := opts.TrainingSelector
	if selector == "" {
		selector = labels.DefaultTrainingSelector
	}
	pods, err := ListTrainingPodsOnNode(ctx, client, nodeName, selector, opts.TargetNamespaces)
	if err != nil {
		return ActionNone, "", err
	}
	if _, err := EvictPods(ctx, client, pods); err != nil {
		return ActionNone, "", err
	}
	if err := SetHealingState(ctx, client, nodeName, labels.StateEvicted); err != nil {
		return ActionNone, "", err
	}
	recordEvent(ctx, client, nodeName, opts, "evict", fmt.Sprintf("evicted %d training pod(s)", len(pods)))
	return ActionEvict, labels.StateEvicted, nil
}

func recordEvent(ctx context.Context, client kubernetes.Interface, nodeName string, opts Options, action, msg string) {
	if opts.SkipEvents || opts.DryRun {
		return
	}
	_ = RecordHealingEvent(ctx, client, nodeName, action, msg)
}
