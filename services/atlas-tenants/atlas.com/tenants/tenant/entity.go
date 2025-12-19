package tenant

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a tenant in the database
type Entity struct {
	gorm.Model
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string    `gorm:"not null"`
	Region       string    `gorm:"not null"`
	MajorVersion uint16    `gorm:"not null"`
	MinorVersion uint16    `gorm:"not null"`
}

// TableName overrides the table name
func (Entity) TableName() string {
	return "tenants"
}

// MigrateEntities creates the tenant table in the database
func MigrateEntities(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}