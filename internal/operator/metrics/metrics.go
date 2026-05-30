package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	operatorUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "operator_up",
		Help: "1 if the operator process is running.",
	})
	healingActions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "healing_actions_total",
		Help: "Healing actions executed.",
	}, []string{"action", "result", "node", "dry_run"})
	healingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "healing_duration_seconds",
		Help:    "Duration of a full handle-node attempt.",
		Buckets: prometheus.DefBuckets,
	}, []string{"node", "result"})
	healingLastSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "healing_last_success_timestamp",
		Help: "Unix timestamp of the last successful healing completion.",
	})
	healingRecovery = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "healing_recovery_total",
		Help: "Job reschedule verification outcomes after eviction.",
	}, []string{"node", "result"})
)

func init() {
	prometheus.MustRegister(operatorUp, healingActions, healingDuration, healingLastSuccess, healingRecovery)
	operatorUp.Set(1)
}

// StartServer exposes Prometheus metrics on listenAddr (e.g. ":8080").
func StartServer(listenAddr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &http.Server{Addr: listenAddr, Handler: mux}
}

// RecordStep increments action counters (skipped in dry-run).
func RecordStep(action, result, node string, dryRun bool) {
	if dryRun {
		return
	}
	healingActions.WithLabelValues(action, result, node, "false").Inc()
}

// ObserveHandleNode records end-to-end handle duration.
func ObserveHandleNode(node, result string, dryRun bool, d time.Duration) {
	if dryRun {
		return
	}
	healingDuration.WithLabelValues(node, result).Observe(d.Seconds())
}

// MarkSuccess sets last success timestamp.
func MarkSuccess() {
	healingLastSuccess.Set(float64(time.Now().Unix()))
}

// RecordRecovery increments verify/reschedule outcome counters.
func RecordRecovery(node, result string, dryRun bool) {
	if dryRun {
		return
	}
	healingRecovery.WithLabelValues(node, result).Inc()
}
