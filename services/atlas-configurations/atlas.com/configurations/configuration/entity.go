package configuration

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TypeChannelService   = "channel-service"
	TypeCharacterFactory = "character-factory"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	ServiceId uuid.UUID       `gorm:"not null"`
	Type      string          `gorm:"not null"`
	Data      json.RawMessage `gorm:"type:json;not null"`
}

func (e Entity) TableName() string {
	return "configurations"
}
