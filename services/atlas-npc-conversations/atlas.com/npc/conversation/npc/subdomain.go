package npc

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
var _ seeder.Subdomain[RestModel, Model] = NpcConversationSubdomain{}

// NpcConversationSubdomain implements seeder.Subdomain for NPC conversations.
type NpcConversationSubdomain struct{}

func (NpcConversationSubdomain) Name() string { return "npc.conversation" }
func (NpcConversationSubdomain) Path() string { return "npc-conversations/npc" }
func (NpcConversationSubdomain) Type() string { return "npc-conversation" }
func (NpcConversationSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^npc-(\d+)\.json$`)
}

func (NpcConversationSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (NpcConversationSubdomain) Decode(payload []byte) (RestModel, error) {
	var rm RestModel
	if err := json.Unmarshal(payload, &rm); err != nil {
		return RestModel{}, fmt.Errorf("npc-conversations: decode: %w", err)
	}
	return rm, nil
}

func (NpcConversationSubdomain) Build(t tenant.Model, _ string, rm RestModel) ([]Model, error) {
	_ = t // tenant tracked via GORM context; not embedded in the domain model
	m, err := Extract(rm)
	if err != nil {
		return nil, fmt.Errorf("npc-conversations: build: %w", err)
	}
	return []Model{m}, nil
}

func (NpcConversationSubdomain) BulkCreate(db *gorm.DB, models []Model) error {
	if len(models) == 0 {
		return nil
	}

	tenantId := extractNpcTenantId(db)
	entities := make([]Entity, 0, len(models))
	for _, m := range models {
		e, err := ToEntity(m, tenantId)
		if err != nil {
			return err
		}
		e.ID = uuid.New()
		entities = append(entities, e)
	}
	return db.Create(&entities).Error
}

func (NpcConversationSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractNpcTenantId retrieves the tenant ID embedded in the GORM context.
func extractNpcTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
