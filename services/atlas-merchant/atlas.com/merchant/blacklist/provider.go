package blacklist

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByShopId(shopId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		if err := db.Where("shop_id = ?", shopId).Order("name ASC").Find(&results).Error; err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

func existsByShopIdAndName(shopId uuid.UUID, name string) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		var count int64
		if err := db.Model(&Entity{}).Where("shop_id = ? AND name = ?", shopId, name).Count(&count).Error; err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(count > 0)
	}
}
