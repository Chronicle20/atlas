package script

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// getByScriptNamePagedProvider returns a provider for retrieving one page of
// map scripts by name (all types)
func getByScriptNamePagedProvider(scriptName string, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("script_name = ?", scriptName), page)
	}
}

// getAllPagedProvider returns a provider for retrieving one page of map
// scripts for a tenant
func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}
