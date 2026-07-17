package item

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getByGachaponIdPagedProvider(gachaponId string, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(&entity{GachaponId: gachaponId}), page)
	}
}

// getByGachaponId returns every item for the given gachapon, regardless of
// tier. Modeled on getByGachaponIdAndTier, minus the tier filter.
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

func getByGachaponIdAndTierPagedProvider(gachaponId string, tier string, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(&entity{GachaponId: gachaponId, Tier: tier}), page)
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, e.ID).
		SetGachaponId(e.GachaponId).
		SetItemId(e.ItemId).
		SetQuantity(e.Quantity).
		SetTier(e.Tier).
		SetWeight(e.Weight).
		Build()
}
