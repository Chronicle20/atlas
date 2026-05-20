package script

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[jsonMapScript, MapScript] = OnFirstUserEnterSubdomain{}

// OnFirstUserEnterSubdomain implements seeder.Subdomain for onFirstUserEnter map scripts.
type OnFirstUserEnterSubdomain struct{}

const scriptTypeOnFirstUserEnter = "onFirstUserEnter"

func (OnFirstUserEnterSubdomain) Name() string { return scriptTypeOnFirstUserEnter }
func (OnFirstUserEnterSubdomain) Path() string { return "map-actions/onFirstUserEnter" }
func (OnFirstUserEnterSubdomain) Type() string { return "map-action" }
func (OnFirstUserEnterSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^map-(.+)\.json$`)
}

func (OnFirstUserEnterSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllByType(db, scriptTypeOnFirstUserEnter)
}

func (OnFirstUserEnterSubdomain) Decode(payload []byte) (jsonMapScript, error) {
	var attrs jsonMapScript
	if err := json.Unmarshal(payload, &attrs); err != nil {
		return jsonMapScript{}, fmt.Errorf("onFirstUserEnter: decode attributes: %w", err)
	}
	return attrs, nil
}

func (OnFirstUserEnterSubdomain) Build(t tenant.Model, entityID string, attrs jsonMapScript) ([]MapScript, error) {
	return buildScripts(t.Id(), scriptTypeOnFirstUserEnter, entityID, attrs)
}

func (OnFirstUserEnterSubdomain) BulkCreate(db *gorm.DB, models []MapScript) error {
	tenantId := extractTenantId(db)
	return BulkCreate(db, tenantId, models)
}

func (OnFirstUserEnterSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Where("script_type = ?", scriptTypeOnFirstUserEnter).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
