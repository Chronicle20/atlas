package item

import (
	"atlas-gachapons/gachapon"
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[gachapon.GachaponAttributes, Model] = Subdomain{}

// Subdomain implements seeder.Subdomain for the gachapon_items table.
// It reads the same catalog files as the gachapon subdomain (one per named gachapon)
// but extracts the items embedded in attributes.items.
type Subdomain struct{}

func (Subdomain) Name() string { return "items" }
func (Subdomain) Path() string { return "gachapons" }
func (Subdomain) Type() string { return "gachapon" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^gachapon-(.+)\.json$`)
}

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllForTenant(db)
}

func (Subdomain) Decode(payload []byte) (gachapon.GachaponAttributes, error) {
	return gachapon.Subdomain{}.Decode(payload)
}

func (Subdomain) Build(t tenant.Model, entityID string, attrs gachapon.GachaponAttributes) ([]Model, error) {
	models := make([]Model, 0, len(attrs.Items))
	for i, it := range attrs.Items {
		m, err := NewBuilder(t.Id(), 0).
			SetGachaponId(entityID).
			SetItemId(it.ItemId).
			SetQuantity(it.Quantity).
			SetTier(it.Tier).
			Build()
		if err != nil {
			return nil, fmt.Errorf("item: build model %q[%d]: %w", entityID, i, err)
		}
		models = append(models, m)
	}
	return models, nil
}

func (Subdomain) BulkCreate(db *gorm.DB, models []Model) error {
	return BulkCreateItem(db, models)
}

func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
