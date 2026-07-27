package script

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// getByIdProvider returns a provider for retrieving a reactor script by ID
func getByIdProvider(id uuid.UUID) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("id = ?", id).First(&entity)
			return entity, result.Error
		}
	}
}

// getByReactorIdProvider returns a provider for retrieving a reactor script by reactor ID
func getByReactorIdProvider(reactorId string) func(db *gorm.DB) model.Provider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("reactor_id = ?", reactorId).First(&entity)
			return entity, result.Error
		}
	}
}

// getAllPagedProvider returns a provider for retrieving one page of reactor
// scripts for a tenant
func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}
