// Package main runs the AI-k8s-platform healing operator.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/ai-k8s-platform/core/internal/operator"
	"github.com/ai-k8s-platform/core/internal/operator/config"
	"github.com/ai-k8s-platform/core/internal/operator/metrics"
	"github.com/ai-k8s-platform/core/internal/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	metricsSrv := metrics.StartServer(cfg.MetricsListen)
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			log.Printf("metrics server: %v", err)
		}
	}()
	defer metricsSrv.Shutdown(context.Background())

	k8s, err := newK8sClient()
	if err != nil {
		log.Fatalf("kubernetes client: %v", err)
	}

	var prom prometheus.Client
	if cfg.PrometheusMock {
		prom = prometheus.NewMockClient(cfg.PrometheusMockNodes)
		log.Printf("prometheus mock nodes: %v", cfg.PrometheusMockNodes)
	} else {
		if cfg.PrometheusURL == "" {
			log.Fatal("PROMETHEUS_URL is required when PROMETHEUS_MOCK=false")
		}
		prom = prometheus.NewClient(cfg.PrometheusURL, nil)
	}

	loop := &operator.Loop{K8s: k8s, Prom: prom, Config: cfg}
	log.Printf(`{"msg":"operator started","dry_run":%v,"poll":"%s","metrics":"%s"}`, cfg.HealingDryRun, cfg.PollInterval, cfg.MetricsListen)
	if err := loop.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("operator loop: %v", err)
	}
}

func newK8sClient() (kubernetes.Interface, error) {
	if rc, err := rest.InClusterConfig(); err == nil {
		return kubernetes.NewForConfig(rc)
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}
