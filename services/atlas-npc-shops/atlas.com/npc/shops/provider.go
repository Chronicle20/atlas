package shops

import (
	"atlas-npc/database"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByNpcId returns a provider that gets a shop entity by NPC ID
func getByNpcId(tenantId uuid.UUID, npcId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where(&Entity{TenantId: tenantId, NpcId: npcId}).First(&result).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getAllShops returns a provider that gets all shop entities for a tenant
func getAllShops(tenantId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where(&Entity{TenantId: tenantId}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// existsByNpcId returns a provider that checks if a shop exists for a given NPC ID
func existsByNpcId(tenantId uuid.UUID, npcId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		var count int64
		err := db.Model(&Entity{}).
			Where(&Entity{TenantId: tenantId, NpcId: npcId}).
			Count(&count).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(count > 0)
	}
}
