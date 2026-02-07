package item

import (
	"atlas-gachapons/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByGachaponId(tenantId uuid.UUID, gachaponId string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, GachaponId: gachaponId})
	}
}

func getByGachaponIdAndTier(tenantId uuid.UUID, gachaponId string, tier string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, GachaponId: gachaponId, Tier: tier})
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
