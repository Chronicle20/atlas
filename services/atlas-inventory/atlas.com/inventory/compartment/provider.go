package compartment

import (
	"atlas-inventory/database"
	"github.com/Chronicle20/atlas-constants/inventory"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, Id: id})
	}
}

func getByCharacter(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, &Entity{TenantId: tenantId, CharacterId: characterId})
	}
}

func getByCharacterAndType(tenantId uuid.UUID, characterId uint32, inventoryType inventory.Type) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, CharacterId: characterId, InventoryType: inventoryType})
	}
}
