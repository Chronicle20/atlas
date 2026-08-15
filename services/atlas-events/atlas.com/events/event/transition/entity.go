package transition

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Trigger types identify what caused an occurrence stage transition.
const (
	TriggerTypeOccurrenceCreated = "OCCURRENCE_CREATED"
	TriggerTypeOccurrenceStart   = "OCCURRENCE_START"
	TriggerTypeScheduledWork     = "SCHEDULED_WORK"
	TriggerTypeMonsterKilled     = "MONSTER_KILLED"
	TriggerTypeVoyageArrived     = "VOYAGE_ARRIVED"
)

// Entity is the GORM persistence record for an occurrence stage transition —
// one row per transition, forming the occurrence's history.
type Entity struct {
	ID               uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID         uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	OccurrenceID     uuid.UUID `gorm:"column:occurrence_id;type:uuid;not null;index:ix_trans_occurrence"`
	FromStage        string    `gorm:"column:from_stage"`
	ToStage          string    `gorm:"column:to_stage;not null"`
	OccurredAt       time.Time `gorm:"column:occurred_at;not null"`
	TriggerType      string    `gorm:"column:trigger_type;not null"`
	TriggerReference string    `gorm:"column:trigger_reference"`
}

func (Entity) TableName() string {
	return "event_occurrence_transition"
}

func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Make converts a persistence Entity into a domain Model.
func Make(e Entity) (Model, error) {
	return NewBuilder(e.OccurrenceID, e.ToStage).
		SetId(e.ID).
		SetFromStage(e.FromStage).
		SetOccurredAt(e.OccurredAt).
		SetTrigger(e.TriggerType, e.TriggerReference).
		Build()
}

// ToEntity converts a domain Model into a persistence Entity, stamping the
// tenant id.
func ToEntity(m Model, tenantId uuid.UUID) (Entity, error) {
	return Entity{
		ID:               m.Id(),
		TenantID:         tenantId,
		OccurrenceID:     m.OccurrenceId(),
		FromStage:        m.FromStage(),
		ToStage:          m.ToStage(),
		OccurredAt:       m.OccurredAt(),
		TriggerType:      m.TriggerType(),
		TriggerReference: m.TriggerReference(),
	}, nil
}
