package account

import (
	"atlas-account/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func entityById(tenant tenant.Model, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		where := map[string]interface{}{"tenant_id": tenant.Id(), "id": id}
		var result = Entity{}
		err := db.Where(where).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func entitiesByName(tenant tenant.Model, name string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: tenant.Id(), Name: name}).First(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func allInTenant(tenant tenant.Model) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: tenant.Id()}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func allEntities(db *gorm.DB) model.Provider[[]Entity] {
	var results []Entity
	err := db.Find(&results).Error
	if err != nil {
		return model.ErrorProvider[[]Entity](err)
	}
	return model.FixedProvider[[]Entity](results)
}
