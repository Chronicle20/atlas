package skill

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getByCharacterIdPaged(characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("character_id = ?", characterId), page)
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
