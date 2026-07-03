package pet

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getById(id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).Preload("Excludes").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func getByOwnerId(ownerId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("owner_id = ?", ownerId).Preload("Excludes").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByOwnerIdPaged backs the REST list handler only (GET
// /characters/{characterId}/pets, task-117). getByOwnerId above stays
// unpaged for the internal business-logic callers (spawn/despawn/hunger
// evaluation) that genuinely need every pet a character owns.
func getByOwnerIdPaged(ownerId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("owner_id = ?", ownerId).Preload("Excludes"), page)
	}
}
