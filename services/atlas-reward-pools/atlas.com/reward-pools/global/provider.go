package global

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getAllPagedProvider(page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db, page)
	}
}

func getByTier(tier string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{Tier: tier})
	}
}

func getByTierPagedProvider(tier string, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(&entity{Tier: tier}), page)
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, e.ID).
		SetItemId(e.ItemId).
		SetQuantity(e.Quantity).
		SetTier(e.Tier).
		Build()
}
