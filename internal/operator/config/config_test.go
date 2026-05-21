package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_defaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()
	if cfg.PollInterval != 30*time.Second {
		t.Fatalf("poll = %v", cfg.PollInterval)
	}
	if !cfg.HealingDryRun {
		t.Fatal("expected dry run default true")
	}
	if len(cfg.TargetNamespaces) != 1 || cfg.TargetNamespaces[0] != "ai-training" {
		t.Fatalf("ns = %v", cfg.TargetNamespaces)
	}
}

func TestLoad_mock_nodes(t *testing.T) {
	os.Clearenv()
	os.Setenv("PROMETHEUS_MOCK", "true")
	os.Setenv("PROMETHEUS_MOCK_NODES", "node-1,node-2")
	cfg := Load()
	if !cfg.PrometheusMock || len(cfg.PrometheusMockNodes) != 2 {
		t.Fatalf("mock cfg = %+v", cfg)
	}
}
