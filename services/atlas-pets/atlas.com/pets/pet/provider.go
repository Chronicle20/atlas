package pet

import (
	"atlas-pets/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, id uint64) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, &Entity{TenantId: tenantId, Id: id})
	}
}

func getByOwnerId(tenantId uuid.UUID, ownerId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, &Entity{TenantId: tenantId, OwnerId: ownerId})
	}
}
