package quest

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a quest conversation stored in the database
type Entity struct {
	ID        uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID  uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;index:idx_quest_conversations_tenant_quest,priority:1"`
	QuestID   uint32         `gorm:"column:quest_id;not null;index:idx_quest_conversations_tenant_quest,priority:2"`
	NpcID     uint32         `gorm:"column:npc_id;index"` // Metadata: NPC that gives this quest
	Data      string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the entity
func (Entity) TableName() string {
	return "quest_conversations"
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
		QuestID:   m.QuestId(),
		NpcID:     m.NpcId(),
		Data:      string(jsonData),
		CreatedAt: m.CreatedAt(),
		UpdatedAt: m.UpdatedAt(),
	}, nil
}

// GetByIdProvider returns a provider for retrieving a quest conversation by ID
func GetByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) func(db *gorm.DB) func() (Entity, error) {
	return func(id uuid.UUID) func(db *gorm.DB) func() (Entity, error) {
		return func(db *gorm.DB) func() (Entity, error) {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// GetByQuestIdProvider returns a provider for retrieving a quest conversation by quest ID
func GetByQuestIdProvider(tenantId uuid.UUID) func(questId uint32) func(db *gorm.DB) func() (Entity, error) {
	return func(questId uint32) func(db *gorm.DB) func() (Entity, error) {
		return func(db *gorm.DB) func() (Entity, error) {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND quest_id = ?", tenantId, questId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// GetAllProvider returns a provider for retrieving all quest conversations
func GetAllProvider(tenantId uuid.UUID) func(db *gorm.DB) func() ([]Entity, error) {
	return func(db *gorm.DB) func() ([]Entity, error) {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}

// Create saves a new quest conversation entity to the database
func Create(db *gorm.DB) func(entity Entity) error {
	return func(entity Entity) error {
		return db.Create(&entity).Error
	}
}

// Update updates an existing quest conversation entity in the database
func Update(db *gorm.DB) func(entity Entity) error {
	return func(entity Entity) error {
		return db.Save(&entity).Error
	}
}

// Delete soft-deletes a quest conversation entity from the database
func Delete(tenantId uuid.UUID) func(questId uint32) func(db *gorm.DB) error {
	return func(questId uint32) func(db *gorm.DB) error {
		return func(db *gorm.DB) error {
			return db.Where("tenant_id = ? AND quest_id = ?", tenantId, questId).Delete(&Entity{}).Error
		}
	}
}

// MigrateTable creates or updates the quest_conversations table
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
