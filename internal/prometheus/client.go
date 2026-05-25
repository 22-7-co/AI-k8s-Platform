package prometheus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client queries Prometheus for GPU fault nodes.
type Client interface {
	QueryFaultNodes(ctx context.Context, query string) ([]string, error)
}

type promClient struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a Prometheus API client.
func NewClient(baseURL string, hc *http.Client) Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &promClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  hc,
	}
}

// QueryFaultNodes runs an instant query and returns deduplicated node names.
func (c *promClient) QueryFaultNodes(ctx context.Context, query string) ([]string, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return nil, fmt.Errorf("parse prometheus url: %w", err)
	}
	q := endpoint.Query()
	q.Set("query", query)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read prometheus response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("prometheus http %d: %s", resp.StatusCode, string(body))
	}

	return ParseQueryVectorNodes(body)
}
