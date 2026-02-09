package services

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{}, &HistoryEntity{})
}

type Entity struct {
	Id   uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	Type ServiceType     `gorm:"type:varchar;not null"`
	Data json.RawMessage `gorm:"type:json;not null"`
}

func (e Entity) TableName() string {
	return "services"
}

type HistoryEntity struct {
	Id        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	ServiceId uuid.UUID       `gorm:"type:uuid"`
	Type      ServiceType     `gorm:"type:varchar;not null"`
	Data      json.RawMessage `gorm:"type:json;not null"`
	CreatedAt time.Time       `gorm:"type:timestamp;not null"`
}

func (e HistoryEntity) TableName() string {
	return "service_history"
}
