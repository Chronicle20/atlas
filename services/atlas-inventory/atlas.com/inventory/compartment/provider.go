package compartment

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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
		return database.SliceQuery[Entity](db.Where("character_id = ?", characterId), &Entity{})
	}
}

func getByCharacterAndType(characterId uint32, inventoryType inventory.Type) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db.Where("character_id = ? AND inventory_type = ?", characterId, inventoryType), &Entity{})
	}
}
