package compartment

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider retrieves a compartment by ID
func getByIdProvider(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

// getByAccountIdAndTypeProvider retrieves a compartment by account ID and type
func getByAccountIdAndTypeProvider(accountId uint32) func(type_ CompartmentType) database.EntityProvider[Entity] {
	return func(type_ CompartmentType) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("account_id = ? AND type = ?", accountId, type_).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getAllByAccountIdProvider retrieves all compartments for an account
func getAllByAccountIdProvider(accountId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("account_id = ?", accountId).Find(&entities)
			return entities, result.Error
		}
	}
}
