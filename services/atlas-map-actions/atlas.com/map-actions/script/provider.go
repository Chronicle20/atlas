package script

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a map script by ID
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

// getByScriptNameAndTypeProvider returns a provider for retrieving a map script by name and type
func getByScriptNameAndTypeProvider(tenantId uuid.UUID) func(scriptName string) func(scriptType string) func(db *gorm.DB) model.Provider[Entity] {
	return func(scriptName string) func(scriptType string) func(db *gorm.DB) model.Provider[Entity] {
		return func(scriptType string) func(db *gorm.DB) model.Provider[Entity] {
			return func(db *gorm.DB) model.Provider[Entity] {
				return func() (Entity, error) {
					var entity Entity
					result := db.Where("tenant_id = ? AND script_name = ? AND script_type = ?", tenantId, scriptName, scriptType).First(&entity)
					return entity, result.Error
				}
			}
		}
	}
}

// getByScriptNameProvider returns a provider for retrieving map scripts by name (all types)
func getByScriptNameProvider(tenantId uuid.UUID) func(scriptName string) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(scriptName string) func(db *gorm.DB) model.Provider[[]Entity] {
		return func(db *gorm.DB) model.Provider[[]Entity] {
			return func() ([]Entity, error) {
				var entities []Entity
				result := db.Where("tenant_id = ? AND script_name = ?", tenantId, scriptName).Find(&entities)
				return entities, result.Error
			}
		}
	}
}

// getAllProvider returns a provider for retrieving all map scripts for a tenant
func getAllProvider(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ?", tenantId).Find(&entities)
			return entities, result.Error
		}
	}
}
