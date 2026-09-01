package reagent

import (
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db, page)
	}
}

func getAllProvider() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

func getByItemId(itemId item.Id) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{ReagentItemId: uint32(itemId)})
	}
}
