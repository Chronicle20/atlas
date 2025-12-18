package tenant

import (
	"atlas-tenants/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetByIdProvider returns a provider for a tenant by ID
func GetByIdProvider(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, map[string]interface{}{"id": id})
	}
}

// GetAllProvider returns a provider for all tenants
func GetAllProvider() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, map[string]interface{}{})
	}
}
