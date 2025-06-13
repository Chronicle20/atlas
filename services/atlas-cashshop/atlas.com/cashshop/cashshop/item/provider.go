package item

import (
	"atlas-cashshop/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byIdEntityProvider(tenantId uuid.UUID, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where(&Entity{TenantId: tenantId, Id: id}).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func byCashIdEntityProvider(tenantId uuid.UUID, cashId int64) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var result []Entity
		err := db.Where(&Entity{TenantId: tenantId, CashId: cashId}).Find(&result).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](result)
	}
}
