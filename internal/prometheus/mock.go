package prometheus

import "context"

type mockClient struct {
	nodes []string
}

// NewMockClient returns a client that always returns the configured nodes (PROMETHEUS_MOCK).
func NewMockClient(nodes []string) Client {
	cp := append([]string(nil), nodes...)
	return &mockClient{nodes: cp}
}

func (m *mockClient) QueryFaultNodes(_ context.Context, _ string) ([]string, error) {
	if len(m.nodes) == 0 {
		return nil, nil
	}
	out := append([]string(nil), m.nodes...)
	return out, nil
}
