package asset

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByIdProvider(id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

func getByCompartmentIdProvider(compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("compartment_id = ?", compartmentId).Find(&entities)
			return entities, result.Error
		}
	}
}

func byCashIdProvider(cashId int64) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var result []Entity
			err := db.Where("cash_id = ?", cashId).Find(&result).Error
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}
}
