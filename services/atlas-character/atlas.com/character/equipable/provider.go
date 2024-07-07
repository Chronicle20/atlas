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

func getBySlot(tenantId uuid.UUID, characterId uint32, slot int16) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		var results entity
		err := db.Table("equipables").
			Select("equipables.*").
			Joins("left join inventory on equipables.tenant_id = inventory.tenant_id AND equipables.inventory_id = inventory.id").
			Where("equipables.tenant_id = ? AND equipables.slot = ? AND inventory.character_id = ?", tenantId, slot, characterId).
			First(&results).
			Error
		if err != nil {
			return model.ErrorProvider[entity](err)
		}
		return model.FixedProvider[entity](results)
	}
}
