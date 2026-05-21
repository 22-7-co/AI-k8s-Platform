package operator

import (
	"context"
	"os"
	"time"

	"github.com/ai-k8s-platform/core/internal/controller"
	"github.com/ai-k8s-platform/core/internal/healing"
	"github.com/ai-k8s-platform/core/internal/operator/config"
	"github.com/ai-k8s-platform/core/internal/operator/logging"
	"github.com/ai-k8s-platform/core/internal/prometheus"
	"github.com/ai-k8s-platform/core/pkg/labels"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Loop runs the PromQL polling and healing orchestration.
type Loop struct {
	K8s    kubernetes.Interface
	Prom   prometheus.Client
	Config config.Config
}

// Run starts the polling loop until ctx is cancelled.
func (l *Loop) Run(ctx context.Context) error {
	ticker := time.NewTicker(l.Config.PollInterval)
	defer ticker.Stop()

	for {
		if err := l.tick(ctx); err != nil {
			logging.Info("tick error", logging.Fields{Result: "err", Error: err.Error(), PromQL: l.Config.PrometheusQuery})
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (l *Loop) tick(ctx context.Context) error {
	nodes, err := l.Prom.QueryFaultNodes(ctx, l.Config.PrometheusQuery)
	if err != nil {
		return err
	}
	for _, nodeName := range nodes {
		if err := l.handleNode(ctx, nodeName); err != nil {
			logging.Info("handle node failed", logging.Fields{
				Node: nodeName, Result: "err", Error: err.Error(), PromQL: l.Config.PrometheusQuery,
			})
		}
	}
	return nil
}

func (l *Loop) handleNode(ctx context.Context, nodeName string) error {
	node, err := l.K8s.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if healing.InCooldown(node, l.Config.HealingCooldown) {
		logging.Info("skip node in cooldown", logging.Fields{Node: nodeName, Result: "skip"})
		return nil
	}

	actionID := uuid.NewString()
	opts := healing.Options{
		DryRun:           l.Config.HealingDryRun,
		TargetNamespaces: l.Config.TargetNamespaces,
		TrainingSelector: l.Config.TrainingPodLabelSelector,
	}

	for step := 0; step < 8; step++ {
		action, state, err := healing.AdvanceHealing(ctx, l.K8s, nodeName, opts)
		logging.Info("healing step", logging.Fields{
			ActionID: actionID,
			Node:     nodeName,
			Action:   string(action),
			DryRun:   opts.DryRun,
			Result:   state,
			Error:    errString(err),
		})
		if err != nil {
			return err
		}
		if action == healing.ActionNone {
			if state == labels.StateEvicted && !opts.DryRun {
				return l.afterEvicted(ctx, nodeName, actionID)
			}
			break
		}
		if opts.DryRun {
			break
		}
		if state == labels.StateEvicted {
			return l.afterEvicted(ctx, nodeName, actionID)
		}
	}
	return nil
}

func (l *Loop) afterEvicted(ctx context.Context, nodeName, actionID string) error {
	jobName := getenvOr("TRAINING_JOB_NAME", "training-job")
	namespace := getenvOr("TRAINING_JOB_NAMESPACE", "ai-training")
	if err := controller.WaitForReschedule(ctx, l.K8s, jobName, namespace, nodeName, l.Config.RescheduleTimeout); err != nil {
		logging.Info("verify reschedule failed", logging.Fields{
			ActionID: actionID, Node: nodeName, Action: "verify", Result: "err", Error: err.Error(),
		})
		return err
	}
	if err := healing.RecordHealingEvent(ctx, l.K8s, nodeName, "verify", "training pod running on healthy node"); err != nil {
		return err
	}
	if err := healing.MarkCompleted(ctx, l.K8s, nodeName); err != nil {
		return err
	}
	logging.Info("healing completed", logging.Fields{
		ActionID: actionID, Node: nodeName, Action: "verify", Result: "ok",
	})
	return nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
