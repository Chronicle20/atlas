package pet

import (
	"atlas-pets/database"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where(&Entity{TenantId: tenantId, Id: id}).Preload("Excludes").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func getByOwnerId(tenantId uuid.UUID, ownerId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: tenantId, OwnerId: ownerId}).Preload("Excludes").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
