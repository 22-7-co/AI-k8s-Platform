package prometheus

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ai-k8s-platform/core/pkg/metrics"
)

type queryResponse struct {
	Status    string     `json:"status"`
	ErrorType string     `json:"errorType,omitempty"`
	Error     string     `json:"error,omitempty"`
	Data      queryData  `json:"data"`
}

type queryData struct {
	ResultType string       `json:"resultType"`
	Result     []vectorItem `json:"result"`
}

type vectorItem struct {
	Metric map[string]string `json:"metric"`
}

// ParseQueryVectorNodes extracts unique node names from a Prometheus instant query JSON body.
func ParseQueryVectorNodes(body []byte) ([]string, error) {
	var resp queryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s %s", resp.ErrorType, resp.Error)
	}
	if resp.Data.ResultType != "vector" {
		return nil, fmt.Errorf("unsupported resultType %q", resp.Data.ResultType)
	}

	seen := make(map[string]struct{})
	var nodes []string
	for _, item := range resp.Data.Result {
		node, ok := item.Metric[metrics.LabelNode]
		if !ok || node == "" {
			continue
		}
		if _, dup := seen[node]; dup {
			continue
		}
		seen[node] = struct{}{}
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes, nil
}
