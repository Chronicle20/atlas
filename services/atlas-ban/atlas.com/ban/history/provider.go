package history

import (
	"atlas-ban/database"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func entitiesByAccountId(t tenant.Model, accountId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: t.Id(), AccountId: accountId}).Order("created_at desc").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func entitiesByIP(t tenant.Model, ip string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: t.Id(), IPAddress: ip}).Order("created_at desc").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func entitiesByHWID(t tenant.Model, hwid string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: t.Id(), HWID: hwid}).Order("created_at desc").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func entitiesByTenant(t tenant.Model) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: t.Id()}).Order("created_at desc").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}
