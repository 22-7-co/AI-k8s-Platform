package logging

import (
	"log"
)

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

// Info logs a healing step.
func Info(msg string, f Fields) {
	log.Printf("level=info msg=%q action_id=%s node=%s action=%s dry_run=%v result=%s error=%s xid=%s promql=%s",
		msg, f.ActionID, f.Node, f.Action, f.DryRun, f.Result, f.Error, f.XID, f.PromQL)
}
