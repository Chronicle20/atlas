package script

import (
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[jsonMapScript, MapScript] = OnUserEnterSubdomain{}

// OnUserEnterSubdomain implements seeder.Subdomain for onUserEnter map scripts.
type OnUserEnterSubdomain struct{}

const scriptTypeOnUserEnter = "onUserEnter"

func (OnUserEnterSubdomain) Name() string { return scriptTypeOnUserEnter }
func (OnUserEnterSubdomain) Path() string { return "map-actions/onUserEnter" }
func (OnUserEnterSubdomain) Type() string { return "map-action" }
func (OnUserEnterSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^map-(.+)\.json$`)
}

func (OnUserEnterSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllByType(db, scriptTypeOnUserEnter)
}

func (OnUserEnterSubdomain) Decode(payload []byte) (jsonMapScript, error) {
	var attrs jsonMapScript
	if err := seeder.DecodeAttributes(payload, &attrs); err != nil {
		return jsonMapScript{}, fmt.Errorf("onUserEnter: decode attributes: %w", err)
	}
	return attrs, nil
}

func (OnUserEnterSubdomain) Build(t tenant.Model, entityID string, attrs jsonMapScript) ([]MapScript, error) {
	return buildScripts(t.Id(), scriptTypeOnUserEnter, entityID, attrs)
}

func (OnUserEnterSubdomain) BulkCreate(db *gorm.DB, models []MapScript) error {
	tenantId := extractTenantId(db)
	return BulkCreate(db, tenantId, models)
}

func (OnUserEnterSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Where("script_type = ?", scriptTypeOnUserEnter).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// buildScripts constructs MapScript models from a decoded jsonMapScript.
func buildScripts(tenantId uuid.UUID, scriptType string, entityID string, attrs jsonMapScript) ([]MapScript, error) {
	_ = tenantId // tenant is tracked via GORM context; not embedded in the domain model
	builder := NewMapScriptBuilder().
		SetScriptName(entityID).
		SetScriptType(scriptType).
		SetDescription(attrs.Description)

	for _, jr := range attrs.Rules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return nil, fmt.Errorf("%s: convert rule %q: %w", scriptType, jr.Id, err)
		}
		builder.AddRule(rule)
	}

	return []MapScript{builder.Build()}, nil
}

// extractTenantId retrieves the tenant ID embedded in the GORM context.
func extractTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
