package compartment

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-constants/inventory"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{Id: id})
	}
}

func getByCharacter(characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, &Entity{CharacterId: characterId})
	}
}

func getByCharacterAndType(characterId uint32, inventoryType inventory.Type) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{CharacterId: characterId, InventoryType: inventoryType})
	}
}
