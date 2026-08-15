// Package environments is atlas-configurations' fourth resource: the
// environment list itself. Unlike tenants/templates/services (which carry
// an Environment column scoping each row to one execution environment, see
// environmentcol), this table has no such column — it IS the environment
// list, exactly as the tenant table is the tenant list (design/PRD §3.5).
package environments

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is one execution environment (main, pr-123, ...). Name is the wire
// identity (env.Id); Overrides and the rest of the shape round-trip through
// env.Record on the outbox envelope.
type Entity struct {
	Id        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	Name      string          `gorm:"not null;uniqueIndex"`
	Baseline  string          `gorm:"not null"`
	Namespace string          `gorm:"not null"`
	Tenant    string          `gorm:"not null;default:''"`
	Overrides json.RawMessage `gorm:"type:json;not null"`
	Phase     string          `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "environments"
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
