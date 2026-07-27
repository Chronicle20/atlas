package tenant

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// GetByIdProvider returns a provider for a tenant by ID
func GetByIdProvider(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, map[string]interface{}{"id": id})
	}
}

// getAll returns a paged provider for all tenants
func getAll(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}
