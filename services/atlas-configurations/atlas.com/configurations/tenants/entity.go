package tenants

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{}, &HistoryEntity{})
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

type HistoryEntity struct {
	Id        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	TenantId  uuid.UUID       `gorm:"type:uuid"`
	Data      json.RawMessage `gorm:"type:json;not null"`
	CreatedAt time.Time       `gorm:"type:timestamp;not null"`
}

func (e HistoryEntity) TableName() string {
	return "tenant_history"
}
