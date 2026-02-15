package definition

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entity struct {
	ID        uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID  uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null"`
	QuestID   string         `gorm:"column:quest_id;not null"`
	Data      string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Entity) TableName() string {
	return "definitions"
}

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
		ID:        id,
		TenantID:  tenantId,
		QuestID:   m.QuestId(),
		Data:      string(jsonData),
		CreatedAt: m.CreatedAt(),
		UpdatedAt: m.UpdatedAt(),
	}, nil
}

func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
