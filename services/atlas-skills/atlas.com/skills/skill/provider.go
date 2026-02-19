package skill

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getByCharacterId(characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Where("character_id = ?", characterId).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](result)
	}
}

func getById(characterId uint32, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("character_id = ? AND id = ?", characterId, id).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}
