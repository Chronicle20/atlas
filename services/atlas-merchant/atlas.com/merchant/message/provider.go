package message

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByShopId(shopId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("shop_id = ?", shopId).Order("sent_at ASC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
