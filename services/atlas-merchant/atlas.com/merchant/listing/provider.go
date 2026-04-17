package listing

import (
	"errors"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

type shopCount struct {
	ShopId uuid.UUID `gorm:"column:shop_id"`
	Count  int64     `gorm:"column:count"`
}

func countByShopIds(shopIds []uuid.UUID) database.EntityProvider[map[uuid.UUID]int64] {
	return func(db *gorm.DB) model.Provider[map[uuid.UUID]int64] {
		if len(shopIds) == 0 {
			return model.FixedProvider(make(map[uuid.UUID]int64))
		}
		var counts []shopCount
		err := db.Model(&Entity{}).
			Select("shop_id, COUNT(*) as count").
			Where("shop_id IN ?", shopIds).
			Group("shop_id").
			Find(&counts).Error
		if err != nil {
			return model.ErrorProvider[map[uuid.UUID]int64](err)
		}
		result := make(map[uuid.UUID]int64, len(counts))
		for _, c := range counts {
			result[c.ShopId] = c.Count
		}
		return model.FixedProvider(result)
	}
}
