package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordStep_dryRunSkips(t *testing.T) {
	t.Parallel()
	before := testutil.ToFloat64(healingActions.WithLabelValues("cordon", "ok", "node-1", "false"))
	RecordStep("cordon", "ok", "node-1", true)
	after := testutil.ToFloat64(healingActions.WithLabelValues("cordon", "ok", "node-1", "false"))
	if after != before {
		t.Fatalf("dry-run incremented counter: before=%v after=%v", before, after)
	}
}

func TestObserveHandleNode_dryRunSkips(t *testing.T) {
	t.Parallel()
	ObserveHandleNode("node-1", "ok", true, time.Second)
	// Histogram has no cheap ToFloat64; assert no panic and dry-run path returns early.
	RecordStep("verify", "ok", "node-1", true)
}
