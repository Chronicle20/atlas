package tenants

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
	Id           uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	Region       string          `gorm:"not null"`
	MajorVersion uint16          `gorm:"not null"`
	MinorVersion uint16          `gorm:"not null"`
	Data         json.RawMessage `gorm:"type:json;not null"`
	// Environment scopes this control-plane row to one execution
	// environment (task-232 D5). Never a tenant column: this table exists
	// to provision and serve many tenants.
	Environment string `gorm:"not null;default:''"`
}

func (e Entity) TableName() string {
	return "tenants"
}

type HistoryEntity struct {
	Id        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	TenantId  uuid.UUID       `gorm:"type:uuid"`
	Data      json.RawMessage `gorm:"type:json;not null"`
	CreatedAt time.Time       `gorm:"type:timestamp;not null"`
	// Environment scopes this control-plane row to one execution
	// environment (task-232 D5). Never a tenant column: this table exists
	// to provision and serve many tenants.
	Environment string `gorm:"not null;default:''"`
}

func (e HistoryEntity) TableName() string {
	return "tenant_history"
}
