package document

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id         uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	TenantId   uuid.UUID       `gorm:"not null"`
	Type       string          `gorm:"not null"`
	DocumentId uint32          `gorm:"not null"`
	Content    json.RawMessage `gorm:"type:json;not null"`
}

func (e Entity) TableName() string {
	return "documents"
}
