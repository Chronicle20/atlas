package item

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is one authored item conversation. Unique on (tenant_id, item_id):
// an item has at most one dialogue.
type Entity struct {
	ID         uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID   uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_item_conversations_tenant_item,priority:1"`
	ItemID     uint32         `gorm:"column:item_id;not null;uniqueIndex:idx_item_conversations_tenant_item,priority:2"`
	NpcID      uint32         `gorm:"column:npc_id;index"` // Metadata: NPC the dialogue renders with
	ScriptName string         `gorm:"column:script_name"`  // Metadata: WZ spec/script value
	Data       string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the entity
func (Entity) TableName() string {
	return "item_conversations"
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	var data RestModel
	if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
		return Model{}, err
	}

	data.Id = e.ID
	m, err := Extract(data)
	if err != nil {
		return Model{}, err
	}
	return m, nil
}

// ToEntity converts a Model to an Entity
func ToEntity(m Model, tenantId uuid.UUID) (Entity, error) {
	rm, err := Transform(m)
	if err != nil {
		return Entity{}, err
	}

	jsonData, err := json.Marshal(rm)
	if err != nil {
		return Entity{}, err
	}

	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}

	return Entity{
		ID:         id,
		TenantID:   tenantId,
		ItemID:     m.ItemId(),
		NpcID:      m.NpcId(),
		ScriptName: m.ScriptName(),
		Data:       string(jsonData),
		CreatedAt:  m.CreatedAt(),
		UpdatedAt:  m.UpdatedAt(),
	}, nil
}

// MigrateTable creates or updates the item_conversations table
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
