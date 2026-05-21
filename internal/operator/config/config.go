package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ai-k8s-platform/core/internal/prometheus"
)

// Config holds operator runtime settings (env + defaults).
type Config struct {
	PollInterval            time.Duration
	HealingCooldown         time.Duration
	PrometheusURL           string
	PrometheusQuery         string
	HealingDryRun           bool
	TargetNamespaces        []string
	TrainingPodLabelSelector string
	PrometheusMock          bool
	PrometheusMockNodes     []string
	RescheduleTimeout       time.Duration
}

// Load reads configuration from environment variables.
func Load() Config {
	cfg := Config{
		PollInterval:             durationEnv("POLL_INTERVAL", 30*time.Second),
		HealingCooldown:          durationEnv("HEALING_COOLDOWN", 10*time.Minute),
		PrometheusURL:            strings.TrimRight(os.Getenv("PROMETHEUS_URL"), "/"),
		PrometheusQuery:            envOr("PROMETHEUS_QUERY", prometheus.DefaultFaultQuery),
		HealingDryRun:            boolEnv("HEALING_DRY_RUN", true),
		TargetNamespaces:         splitCSV(os.Getenv("TARGET_NAMESPACES"), "ai-training"),
		TrainingPodLabelSelector: envOr("TRAINING_POD_LABEL_SELECTOR", "ai-k8s-platform.io/training=true"),
		PrometheusMock:           boolEnv("PROMETHEUS_MOCK", false),
		PrometheusMockNodes:      splitCSV(os.Getenv("PROMETHEUS_MOCK_NODES"), ""),
		RescheduleTimeout:        durationEnv("RESCHEDULE_TIMEOUT", 5*time.Minute),
	}
	return cfg
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func boolEnv(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func durationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(v, def string) []string {
	if v == "" {
		if def == "" {
			return nil
		}
		v = def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
