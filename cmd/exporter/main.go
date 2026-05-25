// Package main exports mock GPU metrics for Prometheus scraping.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ai-k8s-platform/core/internal/exporter"
)

func main() {
	listen := os.Getenv("EXPORTER_LISTEN")
	if listen == "" {
		listen = ":9100"
	}

	reg := exporter.NewRegistry()
	if os.Getenv("EXPORTER_SEED_SAMPLE") == "true" {
		reg.SeedSample("node-0", "0", "79")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", reg.Handler())
	mux.HandleFunc("/inject/xid", reg.InjectHandler)

	log.Printf("gpu metrics exporter listening on %s", listen)
	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
