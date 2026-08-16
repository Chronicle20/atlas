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

// ScopingDimension is a zero-value marker (tools/scopeguard's Rule 1,
// task-232 Task 19 fix round 3) declaring, at this entity's own site, that
// it IS the scoping dimension it would otherwise be expected to carry
// (Environment) — the same shape as the tenant table being the tenant
// list. scopeguard requires this marker, an allowlist.txt entry, AND a
// uniquely-constrained natural key (Name) to all agree before excusing
// this entity from the Environment-column requirement; see
// tools/scopeguard/analyzer.go's hasScopingDimensionMarker doc comment for
// why struct shape alone was found to be an insufficient proxy for this
// claim.
func (Entity) ScopingDimension() {}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
