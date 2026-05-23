package config

import (
	"os"
	"testing"
	"time"
)

func TestBackoffDuration_exponential_cap(t *testing.T) {
	os.Clearenv()
	cfg := Load()
	cfg.RetryBackoffBase = time.Second
	cfg.RetryBackoffCap = 5 * time.Minute

	if d := cfg.BackoffDuration(0); d != time.Second {
		t.Fatalf("attempt 0 = %v", d)
	}
	if d := cfg.BackoffDuration(1); d != 2*time.Second {
		t.Fatalf("attempt 1 = %v", d)
	}
	if d := cfg.BackoffDuration(10); d != 5*time.Minute {
		t.Fatalf("attempt 10 = %v, want cap", d)
	}
}

func TestLoad_Retry_env(t *testing.T) {
	os.Clearenv()
	os.Setenv("HEALING_MAX_RETRIES", "3")
	os.Setenv("RETRY_BACKOFF_BASE", "2s")
	cfg := Load()
	if cfg.HealingMaxRetries != 3 || cfg.RetryBackoffBase != 2*time.Second {
		t.Fatalf("cfg = %+v", cfg)
	}
}
