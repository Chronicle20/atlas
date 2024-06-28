package item

import (
	"atlas-character/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, id uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, ID: id})
	}
}

func getForCharacter(tenantId uuid.UUID, inventoryId uint32, itemId uint32) database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId, ItemId: itemId})
	}
}

func getBySlot(tenantId uuid.UUID, inventoryId uint32, slot int16) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId, Slot: slot})
	}
}

func getByInventory(tenantId uuid.UUID, inventoryId uint32) database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId})
	}
}

//func getByEquipmentId(tenantId uuid.UUID, equipmentId uint32) database.EntityProvider[entityInventoryItem] {
//	return func(db *gorm.DB) model.Provider[entityInventoryItem] {
//		return database.Query[entityInventoryItem](db, &entityInventoryItem{TenantId: tenantId, ReferenceId: equipmentId})
//	}
//}

func getItemAttributes(tenantId uuid.UUID, id uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, ID: id})
	}
}

func minFreeSlot(items []Model) int16 {
	slot := int16(1)
	i := 0

	for {
		if i >= len(items) {
			return slot
		} else if slot < items[i].Slot() {
			return slot
		} else if slot == items[i].Slot() {
			slot += 1
			i += 1
		} else if items[i].Slot() <= 0 {
			i += 1
		}
	}
}
