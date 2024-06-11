package account

import (
	"atlas-account/database"
	"atlas-account/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func entityById(tenant tenant.Model, id uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		var result = entity{TenantId: tenant.Id(), ID: id}
		err := db.First(&result).Error
		if err != nil {
			return model.ErrorProvider[entity](err)
		}
		return model.FixedProvider[entity](result)
	}
}

func entitiesByName(tenant tenant.Model, name string) database.EntitySliceProvider[entity] {
	return func(db *gorm.DB) model.SliceProvider[entity] {
		var results []entity
		err := db.Where(&entity{TenantId: tenant.Id(), Name: name}).First(&results).Error
		if err != nil {
			return model.ErrorSliceProvider[entity](err)
		}
		return model.FixedSliceProvider[entity](results)
	}
}
