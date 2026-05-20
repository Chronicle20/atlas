package drop

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[JSONModel, Model] = Subdomain{}

// JSONModel is the decoded shape of the "attributes" field in a continent-drop catalog file.
type JSONModel struct {
	Drops []DropJSON `json:"drops"`
}

// DropJSON is the shape of each entry in the attributes.drops array.
type DropJSON struct {
	ItemID          uint32 `json:"itemId"`
	MinimumQuantity uint32 `json:"minimumQuantity"`
	MaximumQuantity uint32 `json:"maximumQuantity"`
	QuestID         uint32 `json:"questId,omitempty"`
	Chance          uint32 `json:"chance"`
}

// Subdomain implements seeder.Subdomain for the continent_drops table.
type Subdomain struct{}

func (Subdomain) Name() string { return "continent-drop" }
func (Subdomain) Path() string { return "drops/continents" }
func (Subdomain) Type() string { return "continent-drop" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^continent-(-?\d+)\.json$`)
}

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAll(db)
}

func (Subdomain) Decode(payload []byte) (JSONModel, error) {
	var attrs JSONModel
	if err := json.Unmarshal(payload, &attrs); err != nil {
		return JSONModel{}, fmt.Errorf("continent-drop: decode attributes: %w", err)
	}
	return attrs, nil
}

func (Subdomain) Build(t tenant.Model, entityID string, attrs JSONModel) ([]Model, error) {
	continentId64, err := strconv.ParseInt(entityID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("continent-drop: parse entityID %q: %w", entityID, err)
	}
	continentId := int32(continentId64)

	models := make([]Model, 0, len(attrs.Drops))
	for _, d := range attrs.Drops {
		m, err := NewContinentDropBuilder(t.Id(), 0).
			SetContinentId(continentId).
			SetItemId(d.ItemID).
			SetMinimumQuantity(d.MinimumQuantity).
			SetMaximumQuantity(d.MaximumQuantity).
			SetQuestId(d.QuestID).
			SetChance(d.Chance).
			Build()
		if err != nil {
			return nil, fmt.Errorf("continent-drop: build model for continent %d: %w", continentId, err)
		}
		models = append(models, m)
	}
	return models, nil
}

func (Subdomain) BulkCreate(db *gorm.DB, models []Model) error {
	return BulkCreateContinentDrop(db, models)
}

func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
