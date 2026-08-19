package scheduling

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Work types identify what kind of work a scheduled row represents.
const (
	WorkTypeTriggerEvaluation    = "TRIGGER_EVALUATION"
	WorkTypeOccurrenceTransition = "OCCURRENCE_TRANSITION"
)

// States a scheduled work row can be in over its lifetime.
const (
	StatePending    = "PENDING"
	StateProcessing = "PROCESSING"
	StateCompleted  = "COMPLETED"
	StateCancelled  = "CANCELLED"
	StateFailed     = "FAILED"
)

// Model is an immutable representation of a single unit of scheduled event
// work — a row in scheduled_event_work.
//
// TenantId/TenantRegion/TenantMajor/TenantMinor carry the owning tenant's
// full identity, not just its id — the poller claims work across every
// tenant (design §4.2) and must rebuild a tenant.Model for a single claimed
// row before invoking any handler, which a bare id cannot do.
type Model struct {
	id           uuid.UUID
	definitionId uuid.UUID
	occurrenceId uuid.UUID
	theType      string
	context      json.RawMessage
	executeAt    time.Time
	state        string
	attempts     int
	lastError    string
	dedupeKey    string
	tenantId     uuid.UUID
	tenantRegion string
	tenantMajor  uint16
	tenantMinor  uint16
}

func (m Model) Id() uuid.UUID            { return m.id }
func (m Model) DefinitionId() uuid.UUID  { return m.definitionId }
func (m Model) OccurrenceId() uuid.UUID  { return m.occurrenceId }
func (m Model) Type() string             { return m.theType }
func (m Model) Context() json.RawMessage { return m.context }
func (m Model) ExecuteAt() time.Time     { return m.executeAt }
func (m Model) State() string            { return m.state }
func (m Model) Attempts() int            { return m.attempts }
func (m Model) LastError() string        { return m.lastError }
func (m Model) DedupeKey() string        { return m.dedupeKey }
func (m Model) TenantId() uuid.UUID      { return m.tenantId }
func (m Model) TenantRegion() string     { return m.tenantRegion }
func (m Model) TenantMajor() uint16      { return m.tenantMajor }
func (m Model) TenantMinor() uint16      { return m.tenantMinor }
