package global

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

func getByTier(tier string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{Tier: tier})
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, e.ID).
		SetItemId(e.ItemId).
		SetQuantity(e.Quantity).
		SetTier(e.Tier).
		Build()
}
