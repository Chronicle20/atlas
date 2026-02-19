package item

import (
	database "github.com/Chronicle20/atlas-database"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getByGachaponId(gachaponId string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{GachaponId: gachaponId})
	}
}

func getByGachaponIdAndTier(gachaponId string, tier string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{GachaponId: gachaponId, Tier: tier})
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, e.ID).
		SetGachaponId(e.GachaponId).
		SetItemId(e.ItemId).
		SetQuantity(e.Quantity).
		SetTier(e.Tier).
		Build()
}
