package inventory

import (
	"atlas-character/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func get(tenantId uuid.UUID, characterId uint32, inventoryType Type) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, CharacterId: characterId, InventoryType: int8(inventoryType)})
	}
}

func getByCharacter(tenantId uuid.UUID, characterId uint32) database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, CharacterId: characterId})
	}
}
