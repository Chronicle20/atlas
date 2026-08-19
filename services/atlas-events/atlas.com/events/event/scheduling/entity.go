package scheduling

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	ID       uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	// TenantRegion/TenantMajor/TenantMinor denormalize the claiming tenant's
	// identity onto the row, mirroring
	// services/atlas-saga-orchestrator/.../saga/entity.go. The poller reads
	// across every tenant (design §4.2, database.WithoutTenantFilter) and must
	// re-enter a tenant-scoped context per claimed row before invoking any
	// handler — TenantID alone cannot rebuild a tenant.Model, so region/major/
	// minor travel with the row instead of requiring a separate tenant lookup.
	TenantRegion      string     `gorm:"column:tenant_region;not null;default:''"`
	TenantMajor       uint16     `gorm:"column:tenant_major;not null;default:0"`
	TenantMinor       uint16     `gorm:"column:tenant_minor;not null;default:0"`
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

// MigrateTable creates the scheduled_event_work table and its partial
// indexes (Task 19). ix_sew_pending_due is the FR-S4 poller hot path: PARTIAL
// on state so the poll cost stays independent of how many COMPLETED rows have
// accumulated (FR-N16), which is why no retention policy is needed here
// (design §15.7). ix_sew_processing_claimed serves the FR-S7 lease reclaim
// sweep. ux_sew_dedupe backs FR-B4 dedup: partial on state IN
// (PENDING,PROCESSING) so a cancelled/failed row does not block a retry, and
// AND dedupe_key <> ” so every OCCURRENCE_TRANSITION row (which opts out of
// dedup with an empty key) does not collide with every other one for the
// same tenant.
func MigrateTable(db *gorm.DB) error {
	if err := db.AutoMigrate(&Entity{}); err != nil {
		return err
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS ix_sew_pending_due ` +
		`ON scheduled_event_work (execute_at) WHERE state = 'PENDING'`).Error; err != nil {
		return err
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS ix_sew_processing_claimed ` +
		`ON scheduled_event_work (claimed_at) WHERE state = 'PROCESSING'`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ux_sew_dedupe ` +
		`ON scheduled_event_work (tenant_id, dedupe_key) ` +
		`WHERE state IN ('PENDING','PROCESSING') AND dedupe_key <> ''`).Error
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
		SetDedupeKey(e.DedupeKey).
		SetTenant(e.TenantID, e.TenantRegion, e.TenantMajor, e.TenantMinor)
	if e.EventOccurrenceID != nil {
		b.SetOccurrenceId(*e.EventOccurrenceID)
	}

	return b.Build()
}

// ToEntity converts a domain Model into a persistence Entity, stamping the
// full tenant identity (id, region, major/minor version) so a cross-tenant
// reader — the poller — can rebuild a tenant.Model for a single claimed row
// without a separate lookup (design §4.2).
func ToEntity(m Model, t tenant.Model) (Entity, error) {
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
		TenantID:          t.Id(),
		TenantRegion:      t.Region(),
		TenantMajor:       t.MajorVersion(),
		TenantMinor:       t.MinorVersion(),
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
