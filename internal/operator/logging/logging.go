package logging

import (
	"log/slog"
	"os"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// Fields carries structured healing log context (§6.3).
type Fields struct {
	ActionID string
	Node     string
	Action   string
	DryRun   bool
	Result   string
	Error    string
	XID      string
	PromQL   string
}

// Info logs a healing step as one JSON line.
func Info(msg string, f Fields) {
	attrs := []any{
		"msg", msg,
		"action_id", f.ActionID,
		"node", f.Node,
		"action", f.Action,
		"dry_run", f.DryRun,
		"result", f.Result,
		"error", f.Error,
		"xid", f.XID,
		"promql", f.PromQL,
	}
	logger.Info(msg, attrs...)
}
