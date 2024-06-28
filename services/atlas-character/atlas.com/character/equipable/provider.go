package equipable

import (
	"atlas-character/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByInventory(tenantId uuid.UUID, inventoryId uint32) database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId})
	}
}
