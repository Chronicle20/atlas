package asset

import (
	"atlas-cashshop/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider retrieves an asset by ID
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

// getByCompartmentIdProvider retrieves all assets for a compartment
func getByCompartmentIdProvider(tenantId uuid.UUID) func(compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(compartmentId uuid.UUID) database.EntityProvider[[]Entity] {
		return func(db *gorm.DB) model.Provider[[]Entity] {
			return func() ([]Entity, error) {
				var entities []Entity
				result := db.Where("compartment_id = ? AND tenant_id = ?", compartmentId, tenantId).Find(&entities)
				return entities, result.Error
			}
		}
	}
}
