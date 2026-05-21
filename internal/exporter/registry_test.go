package exporter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ai-k8s-platform/core/pkg/metrics"
)

func TestRegistry_metrics_and_inject(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	mux := http.NewServeMux()
	mux.Handle("/metrics", reg.Handler())
	mux.HandleFunc("/inject/xid", reg.InjectHandler)

	injectReq := httptest.NewRequest(http.MethodPost, "/inject/xid?node=node-1&gpu_id=0&xid_code=79", nil)
	injectRec := httptest.NewRecorder()
	mux.ServeHTTP(injectRec, injectReq)
	if injectRec.Code != http.StatusNoContent {
		t.Fatalf("inject status = %d", injectRec.Code)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	mux.ServeHTTP(metricsRec, metricsReq)
	body := metricsRec.Body.String()
	if !strings.Contains(body, metrics.MetricGPUXIDErrorsTotal) {
		t.Fatalf("metrics body missing %s", metrics.MetricGPUXIDErrorsTotal)
	}
	if !strings.Contains(body, "node-1") {
		t.Fatalf("metrics body missing node-1")
	}
}
