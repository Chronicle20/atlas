package searchcount

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Entity is a per-tenant, per-world search counter for one item id.
// Tenant-safe PK pattern (FR-12): uuid surrogate PK + unique index on
// (tenant_id, world_id, item_id) — never a bare business-key PK.
type Entity struct {
	gorm.Model
	Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_listing_search_counts_tenant_world_item"`
	WorldId  world.Id  `gorm:"not null;uniqueIndex:idx_listing_search_counts_tenant_world_item"`
	ItemId   uint32    `gorm:"not null;uniqueIndex:idx_listing_search_counts_tenant_world_item"`
	Count    uint64    `gorm:"not null;default:0"`
}

func (e *Entity) TableName() string {
	return "listing_search_counts"
}

func Make(e Entity) (Model, error) {
	return Model{itemId: e.ItemId, count: e.Count}, nil
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
