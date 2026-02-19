package key

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func byCharacterKeyEntityProvider(characterId uint32, key int32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("character_id = ? AND key = ?", characterId, key), &entity{})
	}
}

func byCharacterIdEntityProvider(characterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("character_id = ?", characterId), &entity{})
	}
}
