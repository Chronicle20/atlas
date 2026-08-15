package scheduling

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is the GORM persistence record for one unit of scheduled event
// work. EventOccurrenceID is a plain nullable FK column — the event/occurrence
// package is not imported here; it is a sibling domain owning its own model.
//
// DedupeKey collapses redelivered scheduling requests for the same logical
// work (FR-B4/FR-S8): the unique index ux_sew_dedupe (created in Task 19) is
// partial over state IN (PENDING, PROCESSING), so a cancelled or failed row
// does not block a legitimate retry of the same key, and an empty key opts a
// row out of dedup entirely.
type Entity struct {
	ID                uuid.UUID  `gorm:"primaryKey;column:id;type:uuid"`
	TenantID          uuid.UUID  `gorm:"column:tenant_id;type:uuid;not null"`
	EventDefinitionID uuid.UUID  `gorm:"column:event_definition_id;type:uuid;not null"`
	EventOccurrenceID *uuid.UUID `gorm:"column:event_occurrence_id;type:uuid"`
	Type              string     `gorm:"column:type;not null"`
	Context           string     `gorm:"column:context;type:jsonb;not null;default:'{}'"`
	ExecuteAt         time.Time  `gorm:"column:execute_at;not null"`
	State             string     `gorm:"column:state;not null"`
	ClaimedBy         string     `gorm:"column:claimed_by"`
	ClaimedAt         *time.Time `gorm:"column:claimed_at"`
	Attempts          int        `gorm:"column:attempts;not null;default:0"`
	LastError         string     `gorm:"column:last_error"`
	DedupeKey         string     `gorm:"column:dedupe_key;not null;default:''"`
}

func (Entity) TableName() string {
	return "scheduled_event_work"
}

func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Make converts a persistence Entity into a domain Model.
func Make(e Entity) (Model, error) {
	cfg := json.RawMessage(e.Context)

	b := NewBuilder(e.EventDefinitionID, e.Type).
		SetId(e.ID).
		SetContext(cfg).
		SetExecuteAt(e.ExecuteAt).
		SetState(e.State).
		SetAttempts(e.Attempts).
		SetLastError(e.LastError).
		SetDedupeKey(e.DedupeKey)
	if e.EventOccurrenceID != nil {
		b.SetOccurrenceId(*e.EventOccurrenceID)
	}

	return b.Build()
}

// ToEntity converts a domain Model into a persistence Entity, stamping the
// tenant id.
func ToEntity(m Model, tenantId uuid.UUID) (Entity, error) {
	cfg := m.Context()
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}

	var occurrenceId *uuid.UUID
	if m.OccurrenceId() != uuid.Nil {
		id := m.OccurrenceId()
		occurrenceId = &id
	}

	return Entity{
		ID:                m.Id(),
		TenantID:          tenantId,
		EventDefinitionID: m.DefinitionId(),
		EventOccurrenceID: occurrenceId,
		Type:              m.Type(),
		Context:           string(cfg),
		ExecuteAt:         m.ExecuteAt(),
		State:             m.State(),
		Attempts:          m.Attempts(),
		LastError:         m.LastError(),
		DedupeKey:         m.DedupeKey(),
	}, nil
}
