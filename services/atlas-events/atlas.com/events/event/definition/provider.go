package definition

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getByIdProvider(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, map[string]interface{}{"id": id})
	}
}

func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}

func getByTypeProvider(theType string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, map[string]interface{}{"type": theType})
	}
}

func getEnabledByTypeProvider(theType string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, map[string]interface{}{"type": theType, "enabled": true})
	}
}
