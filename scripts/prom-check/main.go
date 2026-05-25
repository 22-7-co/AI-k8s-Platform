// Command prom-check queries Prometheus for fault nodes (P1 local verification).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ai-k8s-platform/core/internal/prometheus"
)

func main() {
	baseURL := os.Getenv("PROMETHEUS_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}
	query := os.Getenv("PROMETHEUS_QUERY")
	if query == "" {
		query = prometheus.DefaultFaultQuery
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := prometheus.NewClient(baseURL, nil)
	nodes, err := client.QueryFaultNodes(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	if len(nodes) == 0 {
		fmt.Println("no fault nodes")
		return
	}
	fmt.Println("fault nodes:", strings.Join(nodes, ", "))
}
