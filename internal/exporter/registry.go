package exporter

import (
	"net/http"
	"strconv"

	"github.com/ai-k8s-platform/core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry holds mock GPU metrics for development and demos.
type Registry struct {
	reg       *prometheus.Registry
	xidErrors *prometheus.CounterVec
}

// NewRegistry registers gpu_xid_errors_total and returns a Registry.
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()
	xidErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricGPUXIDErrorsTotal,
			Help: "Total GPU XID hardware errors observed on the node.",
		},
		[]string{metrics.LabelNode, metrics.LabelGPUID, metrics.LabelXIDCode},
	)
	reg.MustRegister(xidErrors)
	return &Registry{reg: reg, xidErrors: xidErrors}
}

// Handler returns the Prometheus /metrics handler.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

// InjectXID increments the XID counter for the given labels (dev/CI fault injection).
func (r *Registry) InjectXID(node, gpuID, xidCode string) {
	r.xidErrors.WithLabelValues(node, gpuID, xidCode).Inc()
}

// SeedSample pre-increments a sample series at startup (env/flag bootstrap).
func (r *Registry) SeedSample(node, gpuID, xidCode string) {
	r.InjectXID(node, gpuID, xidCode)
}

// InjectHandler handles POST /inject/xid?node=&gpu_id=&xid_code=.
func (r *Registry) InjectHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := req.URL.Query()
	node := q.Get("node")
	if node == "" {
		http.Error(w, "node is required", http.StatusBadRequest)
		return
	}
	gpuID := q.Get("gpu_id")
	if gpuID == "" {
		gpuID = "0"
	}
	xidCode := q.Get("xid_code")
	if xidCode == "" {
		xidCode = "79"
	}
	if _, err := strconv.Atoi(gpuID); err != nil {
		http.Error(w, "gpu_id must be numeric", http.StatusBadRequest)
		return
	}
	r.InjectXID(node, gpuID, xidCode)
	w.WriteHeader(http.StatusNoContent)
}
