package compartment

import (
	"atlas-cashshop/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider retrieves a compartment by ID
func getByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) database.EntityProvider[Entity] {
	return func(id uuid.UUID) database.EntityProvider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("id = ? AND tenant_id = ?", id, tenantId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getByAccountIdAndTypeProvider retrieves a compartment by account ID and type
func getByAccountIdAndTypeProvider(tenantId uuid.UUID) func(accountId uint32) func(type_ CompartmentType) database.EntityProvider[Entity] {
	return func(accountId uint32) func(type_ CompartmentType) database.EntityProvider[Entity] {
		return func(type_ CompartmentType) database.EntityProvider[Entity] {
			return func(db *gorm.DB) model.Provider[Entity] {
				return func() (Entity, error) {
					var entity Entity
					result := db.Where("account_id = ? AND type = ? AND tenant_id = ?", accountId, type_, tenantId).First(&entity)
					return entity, result.Error
				}
			}
		}
	}
}

// getAllByAccountIdProvider retrieves all compartments for an account
func getAllByAccountIdProvider(tenantId uuid.UUID) func(accountId uint32) database.EntityProvider[[]Entity] {
	return func(accountId uint32) database.EntityProvider[[]Entity] {
		return func(db *gorm.DB) model.Provider[[]Entity] {
			return func() ([]Entity, error) {
				var entities []Entity
				result := db.Where("account_id = ? AND tenant_id = ?", accountId, tenantId).Find(&entities)
				return entities, result.Error
			}
		}
	}
}
