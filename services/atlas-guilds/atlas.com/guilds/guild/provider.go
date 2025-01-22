package guild

import (
	"atlas-guilds/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getAll(tenantId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, &Entity{TenantId: tenantId})
	}
}

func getById(tenantId uuid.UUID, id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, Id: id})
	}
}

func getForName(tenantId uuid.UUID, worldId byte, name string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("tenant_id = ? AND worldId = ? AND LOWER(name) = LOWER(?)", tenantId, worldId, name).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
