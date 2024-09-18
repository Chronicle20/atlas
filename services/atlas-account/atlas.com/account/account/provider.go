package account

import (
	"atlas-account/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func entityById(tenant tenant.Model, id uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		where := map[string]interface{}{"tenant_id": tenant.Id(), "id": id}
		var result = entity{}
		err := db.Where(where).First(&result).Error
		if err != nil {
			return model.ErrorProvider[entity](err)
		}
		return model.FixedProvider[entity](result)
	}
}

func entitiesByName(tenant tenant.Model, name string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(&entity{TenantId: tenant.Id(), Name: name}).First(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider[[]entity](results)
	}
}

func allInTenant(tenant tenant.Model) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(&entity{TenantId: tenant.Id()}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider[[]entity](results)
	}
}

func allEntities(db *gorm.DB) model.Provider[[]entity] {
	var results []entity
	err := db.Find(&results).Error
	if err != nil {
		return model.ErrorProvider[[]entity](err)
	}
	return model.FixedProvider[[]entity](results)
}
