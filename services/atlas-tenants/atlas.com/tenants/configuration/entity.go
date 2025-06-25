package configuration

import (
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a configuration in the database
type Entity struct {
	gorm.Model
	TenantID     uuid.UUID      `gorm:"type:uuid;not null"`
	ResourceName string         `gorm:"not null"`
	ResourceData json.RawMessage `gorm:"type:jsonb;not null"`
}

// TableName overrides the table name
func (Entity) TableName() string {
	return "configurations"
}

// MigrateEntities creates the configuration table in the database
func MigrateEntities(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}