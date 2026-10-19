// Package audit records who did what to which record, satisfying the
// blueprint's requirement to log access/change events for sensitive data
// (patients, billing, clinical records, ...) into the audit_logs
// collection. Every module writes entries through this shared component
// starting with Patients (Phase 2); Phase 9 adds the query/reporting API
// on top of the same collection.
package audit

import "context"

// Entry describes a single auditable event.
type Entry struct {
	ActorID    string
	ActorEmail string
	ActorRole  string
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
	IP         string
}

// Recorder persists audit entries. Implementations must not let a failure
// to record an entry block the operation being audited — callers log a
// warning on error and proceed, the same pattern used for notifications.
type Recorder interface {
	Record(ctx context.Context, entry Entry) error
}

// Actor is the minimal "who is performing this action" identity that
// handlers extract from JWT claims and pass down into service calls that
// need to audit-log. It is defined here (not in the auth module) so every
// module can depend on it without creating a module->module import cycle.
type Actor struct {
	UserID string
	Email  string
	Role   string
}
