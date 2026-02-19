package script

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByIdProvider returns a provider for retrieving a map script by ID
func getByIdProvider(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

// getByScriptNameAndTypeProvider returns a provider for retrieving a map script by name and type
func getByScriptNameAndTypeProvider(scriptName string) func(scriptType string) func(db *gorm.DB) model.Provider[Entity] {
	return func(scriptType string) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("script_name = ? AND script_type = ?", scriptName, scriptType).First(&entity)
				return entity, result.Error
			}
		}
	}
}

// getByScriptNameProvider returns a provider for retrieving map scripts by name (all types)
func getByScriptNameProvider(scriptName string) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("script_name = ?", scriptName).Find(&entities)
			return entities, result.Error
		}
	}
}

// getAllProvider returns a provider for retrieving all map scripts for a tenant
func getAllProvider(db *gorm.DB) model.Provider[[]Entity] {
	return func() ([]Entity, error) {
		var entities []Entity
		result := db.Find(&entities)
		return entities, result.Error
	}
}
