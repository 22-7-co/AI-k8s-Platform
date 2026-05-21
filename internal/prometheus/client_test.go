package prometheus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryFaultNodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       interface{}
		wantNodes  []string
		wantErr    bool
	}{
		{
			name:       "query_returns_fault_nodes",
			statusCode: http.StatusOK,
			body: map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "vector",
					"result": []map[string]interface{}{
						{"metric": map[string]string{"node": "node-2", "gpu_id": "0", "xid_code": "79"}, "value": []interface{}{1.0, "1"}},
						{"metric": map[string]string{"node": "node-1", "gpu_id": "1", "xid_code": "79"}, "value": []interface{}{1.0, "2"}},
					},
				},
			},
			wantNodes: []string{"node-1", "node-2"},
		},
		{
			name:       "query_empty_result",
			statusCode: http.StatusOK,
			body: map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": "vector",
					"result":     []interface{}{},
				},
			},
			wantNodes: nil,
		},
		{
			name:       "query_api_error",
			statusCode: http.StatusOK,
			body: map[string]interface{}{
				"status":    "error",
				"errorType": "bad_data",
				"error":     "invalid query",
			},
			wantErr: true,
		},
		{
			name:       "query_http_failure",
			statusCode: http.StatusInternalServerError,
			body:       map[string]interface{}{"status": "success"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/query" {
					http.NotFound(w, r)
					return
				}
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.body)
			}))
			defer srv.Close()

			client := NewClient(srv.URL, srv.Client())
			got, err := client.QueryFaultNodes(context.Background(), DefaultFaultQuery)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("QueryFaultNodes: %v", err)
			}
			if len(got) == 0 && len(tt.wantNodes) == 0 {
				return
			}
			if strings.Join(got, ",") != strings.Join(tt.wantNodes, ",") {
				t.Fatalf("nodes = %v, want %v", got, tt.wantNodes)
			}
		})
	}
}

func TestMockClient_returns_configured_nodes(t *testing.T) {
	t.Parallel()
	client := NewMockClient([]string{"node-a"})
	got, err := client.QueryFaultNodes(context.Background(), "any")
	if err != nil {
		t.Fatalf("QueryFaultNodes: %v", err)
	}
	if len(got) != 1 || got[0] != "node-a" {
		t.Fatalf("nodes = %v, want [node-a]", got)
	}
}
