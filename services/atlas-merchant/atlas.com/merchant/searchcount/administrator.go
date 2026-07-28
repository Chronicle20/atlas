package searchcount

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// incrementSearchCount is an atomic upsert: first search inserts count=1,
// subsequent searches increment in-place. Conflict target is the unique
// (tenant_id, world_id, item_id) index.
func incrementSearchCount(tenantId uuid.UUID, worldId world.Id, itemId uint32) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		e := &Entity{
			Id:       uuid.New(),
			TenantId: tenantId,
			WorldId:  worldId,
			ItemId:   itemId,
			Count:    1,
		}
		return db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "world_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"count": gorm.Expr("listing_search_counts.count + 1"),
			}),
		}).Create(e).Error
	}
}
