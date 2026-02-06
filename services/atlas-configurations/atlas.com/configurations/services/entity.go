package services

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id   uuid.UUID       `gorm:"type:uuid"`
	Type ServiceType     `gorm:"type:varchar"`
	Data json.RawMessage `gorm:"type:json;not null"`
}

func (e Entity) TableName() string {
	return "services"
}
