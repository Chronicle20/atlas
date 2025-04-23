package item

import (
	"atlas-cashshop/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byIdEntityProvider(tenantId uuid.UUID, itemId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Where(&Entity{TenantId: tenantId, Id: itemId}).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](result)
	}
}
