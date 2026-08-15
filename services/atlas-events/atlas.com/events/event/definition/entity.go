package definition

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is the GORM persistence record for an event definition. Configuration
// is stored as its own jsonb column — rather than the whole model serialized —
// so the generic layer can hand it to a handler without knowing its shape
// (FR-D2), and the scalar columns remain indexable (FR-API7).
type Entity struct {
	ID            uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID      uuid.UUID `gorm:"column:tenant_id;type:uuid;not null"`
	Type          string    `gorm:"column:type;not null;index:ix_def_tenant_type,priority:2"`
	Name          string    `gorm:"column:name;not null"`
	Enabled       bool      `gorm:"column:enabled;not null;default:false"`
	Configuration string    `gorm:"column:configuration;type:jsonb;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
}

func (Entity) TableName() string {
	return "event_definition"
}

func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Make converts a persistence Entity into a domain Model.
func Make(e Entity) (Model, error) {
	cfg := json.RawMessage(e.Configuration)

	return NewBuilder(e.Type, e.Name).
		SetId(e.ID).
		SetEnabled(e.Enabled).
		SetConfiguration(cfg).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}

// ToEntity converts a domain Model into a persistence Entity, stamping the
// tenant id.
func ToEntity(m Model, tenantId uuid.UUID) (Entity, error) {
	cfg := m.Configuration()
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}

	return Entity{
		ID:            m.Id(),
		TenantID:      tenantId,
		Type:          m.Type(),
		Name:          m.Name(),
		Enabled:       m.Enabled(),
		Configuration: string(cfg),
		CreatedAt:     m.CreatedAt(),
		UpdatedAt:     m.UpdatedAt(),
	}, nil
}
