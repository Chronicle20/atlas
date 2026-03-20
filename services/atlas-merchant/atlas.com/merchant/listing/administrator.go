package listing

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createListing(entity *Entity) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		err := db.Create(entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(*entity)
	}
}

func deleteListing(id uuid.UUID) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where("id = ?", id).Delete(&Entity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func updateBundles(id uuid.UUID, bundlesRemaining uint16, quantity uint16, expectedVersion uint32) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		result := db.Model(&Entity{}).
			Where("id = ? AND version = ?", id, expectedVersion).
			Updates(map[string]interface{}{
				"bundles_remaining": bundlesRemaining,
				"quantity":          quantity,
				"version":           expectedVersion + 1,
			})
		if result.Error != nil {
			return model.ErrorProvider[int64](result.Error)
		}
		return model.FixedProvider(result.RowsAffected)
	}
}

func deleteByShopId(shopId uuid.UUID) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where("shop_id = ?", shopId).Delete(&Entity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func decrementDisplayOrderAfter(shopId uuid.UUID, afterOrder uint16) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		result := db.Model(&Entity{}).
			Where("shop_id = ? AND display_order > ?", shopId, afterOrder).
			UpdateColumn("display_order", gorm.Expr("display_order - 1"))
		if result.Error != nil {
			return model.ErrorProvider[int64](result.Error)
		}
		return model.FixedProvider(result.RowsAffected)
	}
}

func updateListingFields(id uuid.UUID, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		quantity := bundleSize * bundleCount
		result := db.Model(&Entity{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"price_per_bundle":  pricePerBundle,
				"bundle_size":       bundleSize,
				"bundles_remaining": bundleCount,
				"quantity":          quantity,
				"version":           gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return model.ErrorProvider[int64](result.Error)
		}
		return model.FixedProvider(result.RowsAffected)
	}
}
