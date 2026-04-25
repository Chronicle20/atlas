package recipe

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents one recipe row, derived from a craftAction state inside an
// NPC conversation.
type Entity struct {
	ID                   uuid.UUID `gorm:"primaryKey;column:id;type:uuid"`
	TenantID             uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;index:idx_recipes_tenant_item,priority:1;index:idx_recipes_tenant_npc,priority:1;uniqueIndex:idx_recipes_tenant_conv_state,priority:1"`
	ConversationID       uuid.UUID `gorm:"column:conversation_id;type:uuid;not null;index:idx_recipes_conversation;uniqueIndex:idx_recipes_tenant_conv_state,priority:2"`
	NpcID                uint32    `gorm:"column:npc_id;not null;index:idx_recipes_tenant_npc,priority:2"`
	StateID              string    `gorm:"column:state_id;type:text;not null;uniqueIndex:idx_recipes_tenant_conv_state,priority:3"`
	ItemID               uint32    `gorm:"column:item_id;not null;index:idx_recipes_tenant_item,priority:2"`
	Materials            string    `gorm:"column:materials;type:jsonb;not null;default:'[]'"`
	MesoCost             uint32    `gorm:"column:meso_cost;not null;default:0"`
	StimulatorID         uint32    `gorm:"column:stimulator_id;not null;default:0"`
	StimulatorFailChance float64   `gorm:"column:stimulator_fail_chance;not null;default:0"`
	CreatedAt            time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name.
func (Entity) TableName() string {
	return "recipes"
}

// MigrateTable creates or updates the recipes table.
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
