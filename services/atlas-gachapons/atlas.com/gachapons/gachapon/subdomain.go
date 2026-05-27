package gachapon

import (
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[GachaponAttributes, Model] = Subdomain{}

// GachaponAttributes is the decoded shape of the "attributes" field in a gachapon catalog file.
// The Items slice is also present in the catalog file (used by the item subdomain).
type GachaponAttributes struct {
	Name           string       `json:"name"`
	NpcIds         []uint32     `json:"npcIds"`
	CommonWeight   uint32       `json:"commonWeight"`
	UncommonWeight uint32       `json:"uncommonWeight"`
	RareWeight     uint32       `json:"rareWeight"`
	Items          []ItemAttrib `json:"items"`
}

// ItemAttrib is the shape of each entry in the attributes.items array.
type ItemAttrib struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
	Tier     string `json:"tier"`
}

// Subdomain implements seeder.Subdomain for the gachapon table.
type Subdomain struct{}

func (Subdomain) Name() string { return "gachapons" }
func (Subdomain) Path() string { return "gachapons" }
func (Subdomain) Type() string { return "gachapon" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^gachapon-(.+)\.json$`)
}

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllForTenant(db)
}

func (Subdomain) Decode(payload []byte) (GachaponAttributes, error) {
	var attrs GachaponAttributes
	if err := seeder.DecodeAttributes(payload, &attrs); err != nil {
		return GachaponAttributes{}, fmt.Errorf("gachapon: decode attributes: %w", err)
	}
	return attrs, nil
}

func (Subdomain) Build(t tenant.Model, entityID string, attrs GachaponAttributes) ([]Model, error) {
	m, err := NewBuilder(t.Id(), entityID).
		SetName(attrs.Name).
		SetNpcIds(attrs.NpcIds).
		SetCommonWeight(attrs.CommonWeight).
		SetUncommonWeight(attrs.UncommonWeight).
		SetRareWeight(attrs.RareWeight).
		Build()
	if err != nil {
		return nil, fmt.Errorf("gachapon: build model %q: %w", entityID, err)
	}
	return []Model{m}, nil
}

func (Subdomain) BulkCreate(db *gorm.DB, models []Model) error {
	return BulkCreateGachapon(db, models)
}

func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
