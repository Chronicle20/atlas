package reagent

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
var _ seeder.Subdomain[ReagentAttributes, Model] = Subdomain{}

// ReagentAttributes is the decoded shape of the "attributes" field in a reagent
// catalog file. The reagent's item id is the file's entity id, not an
// attribute.
type ReagentAttributes struct {
	Stat  string `json:"stat"`
	Value int16  `json:"value"`
}

// Subdomain implements seeder.Subdomain for the reagents table.
type Subdomain struct{}

func (Subdomain) Name() string { return "reagents" }
func (Subdomain) Path() string { return "reagents" }
func (Subdomain) Type() string { return "reagent" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^reagent-(\d+)\.json$`)
}

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllForTenant(db)
}

func (Subdomain) Decode(payload []byte) (ReagentAttributes, error) {
	var attrs ReagentAttributes
	if err := seeder.DecodeAttributes(payload, &attrs); err != nil {
		return ReagentAttributes{}, fmt.Errorf("reagent: decode attributes: %w", err)
	}
	return attrs, nil
}

func (Subdomain) Build(t tenant.Model, entityID string, attrs ReagentAttributes) ([]Model, error) {
	itemId, err := strconv.ParseUint(entityID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("reagent: parse item id %q: %w", entityID, err)
	}
	m, err := NewBuilder(t.Id(), item.Id(itemId)).
		SetStat(attrs.Stat).
		SetValue(attrs.Value).
		Build()
	if err != nil {
		return nil, fmt.Errorf("reagent: build model %q: %w", entityID, err)
	}
	return []Model{m}, nil
}

func (Subdomain) BulkCreate(db *gorm.DB, models []Model) error {
	return BulkCreateReagent(db, models)
}

func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
