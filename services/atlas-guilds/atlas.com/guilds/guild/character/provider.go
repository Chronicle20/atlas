package character

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getById(characterId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("character_id = ?", characterId).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}
