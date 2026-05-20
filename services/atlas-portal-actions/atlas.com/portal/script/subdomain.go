package script

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[jsonPortalScript, PortalScript] = PortalSubdomain{}

// PortalSubdomain implements seeder.Subdomain for portal action scripts.
type PortalSubdomain struct{}

func (PortalSubdomain) Name() string { return "portal-actions" }
func (PortalSubdomain) Path() string { return "portal-actions/portals" }
func (PortalSubdomain) Type() string { return "portal-action" }
func (PortalSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^portal-(.+)\.json$`)
}

func (PortalSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (PortalSubdomain) Decode(payload []byte) (jsonPortalScript, error) {
	var attrs jsonPortalScript
	if err := json.Unmarshal(payload, &attrs); err != nil {
		return jsonPortalScript{}, fmt.Errorf("portal-actions: decode attributes: %w", err)
	}
	return attrs, nil
}

func (PortalSubdomain) Build(t tenant.Model, entityID string, attrs jsonPortalScript) ([]PortalScript, error) {
	_ = t // tenant tracked via GORM context; not embedded in the domain model

	// If the JSON carries a PortalId use it; otherwise fall back to the entity ID from the filename
	portalId := attrs.PortalId
	if portalId == "" {
		portalId = entityID
	}

	builder := NewPortalScriptBuilder().
		SetPortalId(portalId).
		SetMapId(attrs.MapId).
		SetDescription(attrs.Description)

	for _, jr := range attrs.Rules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return nil, fmt.Errorf("portal-actions: convert rule %q: %w", jr.Id, err)
		}
		builder.AddRule(rule)
	}

	return []PortalScript{builder.Build()}, nil
}

func (PortalSubdomain) BulkCreate(db *gorm.DB, models []PortalScript) error {
	if len(models) == 0 {
		return nil
	}

	tenantId := extractPortalTenantId(db)
	entities := make([]Entity, 0, len(models))
	for _, m := range models {
		e, err := ToEntity(m, tenantId)
		if err != nil {
			return err
		}
		e.ID = uuid.New()
		entities = append(entities, e)
	}
	return db.Create(&entities).Error
}

func (PortalSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractPortalTenantId retrieves the tenant ID embedded in the GORM context.
func extractPortalTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
