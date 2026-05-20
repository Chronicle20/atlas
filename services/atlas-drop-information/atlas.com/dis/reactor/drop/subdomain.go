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

// JSONModel is the decoded shape of the "attributes" field in a reactor-drop catalog file.
type JSONModel struct {
	Drops []DropJSON `json:"drops"`
}

// DropJSON is the shape of each entry in the attributes.drops array.
type DropJSON struct {
	ItemID  uint32 `json:"itemId"`
	QuestID uint32 `json:"questId,omitempty"`
	Chance  uint32 `json:"chance"`
}

// Subdomain implements seeder.Subdomain for the reactor_drops table.
type Subdomain struct{}

func (Subdomain) Name() string { return "reactor-drop" }
func (Subdomain) Path() string { return "drops/reactors" }
func (Subdomain) Type() string { return "reactor-drop" }
func (Subdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^reactor-(\d+)\.json$`)
}

func (Subdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return DeleteAll(db)
}

func (Subdomain) Decode(payload []byte) (JSONModel, error) {
	var attrs JSONModel
	if err := json.Unmarshal(payload, &attrs); err != nil {
		return JSONModel{}, fmt.Errorf("reactor-drop: decode attributes: %w", err)
	}
	return attrs, nil
}

func (Subdomain) Build(t tenant.Model, entityID string, attrs JSONModel) ([]Model, error) {
	reactorId64, err := strconv.ParseUint(entityID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("reactor-drop: parse entityID %q: %w", entityID, err)
	}
	reactorId := uint32(reactorId64)

	models := make([]Model, 0, len(attrs.Drops))
	for _, d := range attrs.Drops {
		m, err := NewReactorDropBuilder(t.Id(), 0).
			SetReactorId(reactorId).
			SetItemId(d.ItemID).
			SetQuestId(d.QuestID).
			SetChance(d.Chance).
			Build()
		if err != nil {
			return nil, fmt.Errorf("reactor-drop: build model for reactor %d: %w", reactorId, err)
		}
		models = append(models, m)
	}
	return models, nil
}

func (Subdomain) BulkCreate(db *gorm.DB, models []Model) error {
	return BulkCreateReactorDrop(db, models)
}

func (Subdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}
