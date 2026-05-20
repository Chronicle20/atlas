package global

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
var _ seeder.Subdomain[GlobalPoolAttributes, Model] = Subdomain{}

// GlobalPoolAttributes is the decoded shape of the "attributes" field in the
// global gachapon pool catalog file (gachapons/_global/items.json).
type GlobalPoolAttributes struct {
	Items []GlobalItemAttrib `json:"items"`
}

// GlobalItemAttrib is the shape of each entry in the attributes.items array.
type GlobalItemAttrib struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
	Tier     string `json:"tier"`
}

// Subdomain implements seeder.Subdomain for the global_gachapon_items table.
type Subdomain struct{}

func (Subdomain) Name() string { return "globalItems" }

// Path points directly at the _global subdirectory so the walker lists items.json
// without hitting the underscore-skip logic (the skip applies at walk level, not to
// the directory we pass as the root of the walk).
func (Subdomain) Path() string { return "gachapons/_global" }
func (Subdomain) Type() string { return "gachapon-pool" }

// EntityIDPattern is nil: the entity ID is taken directly from data.id in the file.
func (Subdomain) EntityIDPattern() *regexp.Regexp { return nil }

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAllForTenant(db)
}

func (Subdomain) Decode(payload []byte) (GlobalPoolAttributes, error) {
	var attrs GlobalPoolAttributes
	if err := json.Unmarshal(payload, &attrs); err != nil {
		return GlobalPoolAttributes{}, fmt.Errorf("global: decode attributes: %w", err)
	}
	return attrs, nil
}

func (Subdomain) Build(t tenant.Model, _ string, attrs GlobalPoolAttributes) ([]Model, error) {
	models := make([]Model, 0, len(attrs.Items))
	for i, it := range attrs.Items {
		m, err := NewBuilder(t.Id(), 0).
			SetItemId(it.ItemId).
			SetQuantity(it.Quantity).
			SetTier(it.Tier).
			Build()
		if err != nil {
			return nil, fmt.Errorf("global: build model[%d]: %w", i, err)
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
