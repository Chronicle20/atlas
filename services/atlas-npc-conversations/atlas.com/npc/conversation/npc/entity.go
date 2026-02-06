package npc

import (
	"encoding/json"

	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a conversation tree stored in the database
type Entity struct {
	ID        uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID  uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null"`
	NpcID     uint32         `gorm:"column:npc_id;not null"`
	Data      string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the entity
func (Entity) TableName() string {
	return "conversations"
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	// Parse the JSON data
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

	// Convert the data to JSON
	jsonData, err := json.Marshal(rm)
	if err != nil {
		return Entity{}, err
	}

	// Create entity with ID from model, or generate a new one if nil
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}

	return Entity{
		ID:        id,
		TenantID:  tenantId,
		NpcID:     m.NpcId(),
		Data:      string(jsonData),
		CreatedAt: m.CreatedAt(),
		UpdatedAt: m.UpdatedAt(),
	}, nil
}

// MigrateTable creates or updates the conversations table
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
