package ban

import (
	"atlas-ban/database"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func entityById(t tenant.Model, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where(&Entity{TenantId: t.Id(), ID: id}).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}

func entitiesByTenant(t tenant.Model) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: t.Id()}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func entitiesByType(t tenant.Model, banType BanType) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("tenant_id = ? AND ban_type = ?", t.Id(), byte(banType)).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func activeIPBans(t tenant.Model) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		now := time.Now().Unix()
		err := db.Where("tenant_id = ? AND ban_type = ? AND (permanent = ? OR expires_at > ?)", t.Id(), byte(BanTypeIP), true, now).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}

func activeExactBans(t tenant.Model, banType BanType, value string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		now := time.Now().Unix()
		err := db.Where("tenant_id = ? AND ban_type = ? AND value = ? AND (permanent = ? OR expires_at > ?)", t.Id(), byte(banType), value, true, now).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider[[]Entity](results)
	}
}
