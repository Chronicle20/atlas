package tenants

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id           uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	Region       string          `json:"region"`
	MajorVersion uint16          `json:"majorVersion"`
	MinorVersion uint16          `json:"minorVersion"`
	Data         json.RawMessage `gorm:"type:json;not null"`
}

func (e Entity) TableName() string {
	return "tenants"
}
