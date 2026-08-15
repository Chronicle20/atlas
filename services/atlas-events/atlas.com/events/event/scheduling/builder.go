package scheduling

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder provides fluent construction of scheduled event work models.
type Builder struct {
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

// NewBuilder creates a new builder with the required parameters. State
// defaults to StatePending — every scheduled row starts there.
func NewBuilder(definitionId uuid.UUID, theType string) *Builder {
	return &Builder{
		id:           uuid.New(),
		definitionId: definitionId,
		theType:      theType,
		state:        StatePending,
	}
}

// SetId sets the work row id.
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetOccurrenceId sets the occurrence this work is associated with. Left as
// uuid.Nil for work that predates an occurrence (e.g. a TRIGGER_EVALUATION
// that has not yet decided to start one).
func (b *Builder) SetOccurrenceId(occurrenceId uuid.UUID) *Builder {
	b.occurrenceId = occurrenceId
	return b
}

// SetContext sets the opaque work context payload.
func (b *Builder) SetContext(context json.RawMessage) *Builder {
	b.context = context
	return b
}

// SetExecuteAt sets the timestamp the work becomes eligible to run.
func (b *Builder) SetExecuteAt(executeAt time.Time) *Builder {
	b.executeAt = executeAt
	return b
}

// SetState sets the work row's state.
func (b *Builder) SetState(state string) *Builder {
	b.state = state
	return b
}

// SetAttempts sets the number of times this work has been attempted.
func (b *Builder) SetAttempts(attempts int) *Builder {
	b.attempts = attempts
	return b
}

// SetLastError sets the error recorded from the most recent failed attempt.
func (b *Builder) SetLastError(lastError string) *Builder {
	b.lastError = lastError
	return b
}

// SetDedupeKey sets the key used to collapse redelivered scheduling requests
// for the same logical work. An empty key opts out of dedup entirely.
func (b *Builder) SetDedupeKey(dedupeKey string) *Builder {
	b.dedupeKey = dedupeKey
	return b
}

// SetTenant sets the owning tenant's full identity — id, region, and
// major/minor version — not just its id, so a cross-tenant reader (the
// poller) can rebuild a tenant.Model for this one row without a separate
// lookup (design §4.2).
func (b *Builder) SetTenant(id uuid.UUID, region string, major, minor uint16) *Builder {
	b.tenantId = id
	b.tenantRegion = region
	b.tenantMajor = major
	b.tenantMinor = minor
	return b
}

// Build validates invariants and constructs the final immutable model.
func (b *Builder) Build() (Model, error) {
	if b.definitionId == uuid.Nil {
		return Model{}, errors.New("definitionId is required")
	}
	if b.theType == "" {
		return Model{}, errors.New("type is required")
	}
	if b.executeAt.IsZero() {
		return Model{}, errors.New("executeAt is required")
	}
	if b.context != nil && !json.Valid(b.context) {
		return Model{}, errors.New("context must be valid JSON")
	}

	state := b.state
	if state == "" {
		state = StatePending
	}

	return Model{
		id:           b.id,
		definitionId: b.definitionId,
		occurrenceId: b.occurrenceId,
		theType:      b.theType,
		context:      b.context,
		executeAt:    b.executeAt,
		state:        state,
		attempts:     b.attempts,
		lastError:    b.lastError,
		dedupeKey:    b.dedupeKey,
		tenantId:     b.tenantId,
		tenantRegion: b.tenantRegion,
		tenantMajor:  b.tenantMajor,
		tenantMinor:  b.tenantMinor,
	}, nil
}

// Builder returns a builder initialized with the current model's values.
func (m Model) Builder() *Builder {
	return &Builder{
		id:           m.id,
		definitionId: m.definitionId,
		occurrenceId: m.occurrenceId,
		theType:      m.theType,
		context:      m.context,
		executeAt:    m.executeAt,
		state:        m.state,
		attempts:     m.attempts,
		lastError:    m.lastError,
		dedupeKey:    m.dedupeKey,
		tenantId:     m.tenantId,
		tenantRegion: m.tenantRegion,
		tenantMajor:  m.tenantMajor,
		tenantMinor:  m.tenantMinor,
	}
}
