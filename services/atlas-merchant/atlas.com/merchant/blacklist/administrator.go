package blacklist

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func create(tenantId, shopId uuid.UUID, name string) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		e := &Entity{Id: uuid.New(), TenantId: tenantId, ShopId: shopId, Name: name}
		// Idempotent add: a repeat ban of the same name is a no-op.
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(e).Error; err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(*e)
	}
}

func deleteByShopIdAndName(shopId uuid.UUID, name string) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		if err := db.Where("shop_id = ? AND name = ?", shopId, name).Delete(&Entity{}).Error; err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}
