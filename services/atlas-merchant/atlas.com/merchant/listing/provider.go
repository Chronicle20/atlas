package listing

import (
	"errors"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("listing not found")

func getByShopId(shopId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("shop_id = ?", shopId).Order("display_order ASC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

func getByShopIdAndDisplayOrder(shopId uuid.UUID, displayOrder uint16) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("shop_id = ? AND display_order = ?", shopId, displayOrder).First(&result).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

func countByShopId(shopId uuid.UUID) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		var count int64
		err := db.Model(&Entity{}).Where("shop_id = ?", shopId).Count(&count).Error
		if err != nil {
			return model.ErrorProvider[int64](err)
		}
		return model.FixedProvider(count)
	}
}
