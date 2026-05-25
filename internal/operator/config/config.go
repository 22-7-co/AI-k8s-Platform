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
	PollInterval             time.Duration
	HealingCooldown          time.Duration
	PrometheusURL            string
	PrometheusQuery          string
	HealingDryRun            bool
	TargetNamespaces         []string
	TrainingPodLabelSelector string
	PrometheusMock            bool
	PrometheusMockNodes       []string
	RescheduleTimeout        time.Duration
	HealingMaxRetries        int
	RetryBackoffBase         time.Duration
	RetryBackoffCap          time.Duration
	MetricsListen            string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		PollInterval:             durationEnv("POLL_INTERVAL", 30*time.Second),
		HealingCooldown:          durationEnv("HEALING_COOLDOWN", 10*time.Minute),
		PrometheusURL:            strings.TrimRight(os.Getenv("PROMETHEUS_URL"), "/"),
		PrometheusQuery:          envOr("PROMETHEUS_QUERY", prometheus.DefaultFaultQuery),
		HealingDryRun:            boolEnv("HEALING_DRY_RUN", true),
		TargetNamespaces:         splitCSV(os.Getenv("TARGET_NAMESPACES"), "ai-training"),
		TrainingPodLabelSelector: envOr("TRAINING_POD_LABEL_SELECTOR", "ai-k8s-platform.io/training=true"),
		PrometheusMock:           boolEnv("PROMETHEUS_MOCK", false),
		PrometheusMockNodes:      splitCSV(os.Getenv("PROMETHEUS_MOCK_NODES"), ""),
		RescheduleTimeout:        durationEnv("RESCHEDULE_TIMEOUT", 5*time.Minute),
		HealingMaxRetries:        intEnv("HEALING_MAX_RETRIES", 5),
		RetryBackoffBase:         durationEnv("RETRY_BACKOFF_BASE", time.Second),
		RetryBackoffCap:          durationEnv("RETRY_BACKOFF_CAP", 5*time.Minute),
		MetricsListen:            envOr("METRICS_LISTEN", ":8080"),
	}
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

func intEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
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

// BackoffDuration returns exponential backoff for attempt (0-based), capped.
func (c Config) BackoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return c.RetryBackoffBase
	}
	d := c.RetryBackoffBase
	for i := 0; i < attempt; i++ {
		d *= 2
		if d >= c.RetryBackoffCap {
			return c.RetryBackoffCap
		}
	}
	return d
}
