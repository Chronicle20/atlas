package script

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a portal script by ID
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

// getByPortalIdProvider returns a provider for retrieving a portal script by portal ID
func getByPortalIdProvider(tenantId uuid.UUID) func(portalId string) func(db *gorm.DB) model.Provider[Entity] {
	return func(portalId string) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("tenant_id = ? AND portal_id = ?", tenantId, portalId).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getAllProvider returns a provider for retrieving all portal scripts for a tenant
func getAllProvider(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}
