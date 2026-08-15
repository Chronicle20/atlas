package item

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// compile-time assertion
var _ seeder.Subdomain[RestModel, Model] = ItemConversationSubdomain{}

// ItemConversationSubdomain implements seeder.Subdomain for item conversations.
type ItemConversationSubdomain struct{}

func (ItemConversationSubdomain) Name() string { return "item.conversation" }
func (ItemConversationSubdomain) Path() string { return "npc-conversations/items" }
func (ItemConversationSubdomain) Type() string { return "item-conversation" }
func (ItemConversationSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^item-(\d+)\.json$`)
}

func (ItemConversationSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (ItemConversationSubdomain) Decode(payload []byte) (RestModel, error) {
	var rm RestModel
	if err := seeder.DecodeAttributes(payload, &rm); err != nil {
		return RestModel{}, fmt.Errorf("item-conversations: decode: %w", err)
	}
	return rm, nil
}

func (ItemConversationSubdomain) Build(t tenant.Model, _ string, rm RestModel) ([]Model, error) {
	_ = t // tenant tracked via GORM context; not embedded in the domain model
	m, err := Extract(rm)
	if err != nil {
		return nil, fmt.Errorf("item-conversations: build: %w", err)
	}
	return []Model{m}, nil
}

func (ItemConversationSubdomain) BulkCreate(db *gorm.DB, models []Model) error {
	if len(models) == 0 {
		return nil
	}

	tenantId := extractItemTenantId(db)
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

func (ItemConversationSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractItemTenantId retrieves the tenant ID embedded in the GORM context.
func extractItemTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
