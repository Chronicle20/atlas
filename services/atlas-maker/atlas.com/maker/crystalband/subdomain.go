package crystalband

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// compile-time assertion
var _ seeder.Subdomain[CrystalBandAttributes, Model] = Subdomain{}

// CrystalBandAttributes is the decoded shape of the "attributes" field in a
// crystal-band catalog file. The band's minimum level is the file's entity
// id, not an attribute, since (tenant_id, min_level) is the business key.
type CrystalBandAttributes struct {
	MaxLevel      uint32 `json:"maxLevel"`
	CrystalItemId uint32 `json:"crystalItemId"`
	// Count is an Atlas product decision, NOT client-derived data — see the
	// entity field comment.
	Count uint32 `json:"count"`
}

// Subdomain implements seeder.Subdomain for the crystal_bands table.
type Subdomain struct{}

func (Subdomain) Name() string { return "crystalBands" }
func (Subdomain) Path() string { return "crystal-bands" }
func (Subdomain) Type() string { return "crystalBand" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^crystal-band-(\d+)\.json$`)
}

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllForTenant(db)
}

func (Subdomain) Decode(payload []byte) (CrystalBandAttributes, error) {
	var attrs CrystalBandAttributes
	if err := seeder.DecodeAttributes(payload, &attrs); err != nil {
		return CrystalBandAttributes{}, fmt.Errorf("crystalband: decode attributes: %w", err)
	}
	return attrs, nil
}

func (Subdomain) Build(t tenant.Model, entityID string, attrs CrystalBandAttributes) ([]Model, error) {
	minLevel, err := strconv.ParseUint(entityID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("crystalband: parse min level %q: %w", entityID, err)
	}
	m, err := NewBuilder(t.Id()).
		SetMinLevel(uint32(minLevel)).
		SetMaxLevel(attrs.MaxLevel).
		SetCrystalItemId(item.Id(attrs.CrystalItemId)).
		SetCount(attrs.Count).
		Build()
	if err != nil {
		return nil, fmt.Errorf("crystalband: build model %q: %w", entityID, err)
	}
	return []Model{m}, nil
}

func (Subdomain) BulkCreate(db *gorm.DB, models []Model) error {
	return BulkCreateCrystalBand(db, models)
}

func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
