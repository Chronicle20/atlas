package script

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a reactor script by ID
func getByIdProvider(tenantId uuid.UUID) func(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getByReactorIdProvider returns a provider for retrieving a reactor script by reactor ID
func getByReactorIdProvider(tenantId uuid.UUID) func(reactorId string) func(db *gorm.DB) model.Provider[Entity] {
	return func(reactorId string) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND reactor_id = ?", tenantId, reactorId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getAllProvider returns a provider for retrieving all reactor scripts for a tenant
func getAllProvider(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}
