package definition

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
var _ seeder.Subdomain[RestModel, Model] = DefinitionSubdomain{}

// DefinitionSubdomain implements seeder.Subdomain for party-quest definition seed data.
type DefinitionSubdomain struct{}

func (DefinitionSubdomain) Name() string { return "definition.partyquest" }
func (DefinitionSubdomain) Path() string { return "party-quests/definitions" }
func (DefinitionSubdomain) Type() string { return "party-quest-definition" }
func (DefinitionSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^party-quest-(.+)\.json$`)
}

func (DefinitionSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	return deleteAllDefinitions(db)
}

func (DefinitionSubdomain) Decode(payload []byte) (RestModel, error) {
	var rm RestModel
	if err := json.Unmarshal(payload, &rm); err != nil {
		return RestModel{}, fmt.Errorf("definition.partyquest: decode: %w", err)
	}
	return rm, nil
}

func (DefinitionSubdomain) Build(t tenant.Model, _ string, rm RestModel) ([]Model, error) {
	m, err := Extract(rm)
	if err != nil {
		return nil, fmt.Errorf("definition.partyquest: build %s: %w", rm.QuestId, err)
	}
	return []Model{m}, nil
}

func (DefinitionSubdomain) BulkCreate(db *gorm.DB, models []Model) error {
	if len(models) == 0 {
		return nil
	}

	tenantId := extractDefinitionTenantId(db)
	for _, m := range models {
		entity, err := ToEntity(m, tenantId)
		if err != nil {
			return fmt.Errorf("definition.partyquest: to entity %s: %w", m.QuestId(), err)
		}
		entity.ID = uuid.New()
		if result := db.Create(&entity); result.Error != nil {
			return fmt.Errorf("definition.partyquest: create %s: %w", m.QuestId(), result.Error)
		}
	}
	return nil
}

func (DefinitionSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractDefinitionTenantId retrieves the tenant ID embedded in the GORM context.
func extractDefinitionTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
