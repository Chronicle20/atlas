package shops

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

// getByNpcId returns a provider that gets a shop entity by NPC ID
func getByNpcId(npcId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("npc_id = ?", npcId).First(&result).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getAllShopsPaged returns a provider that gets one page of shop entities
// for a tenant. Entity.Id is a single-column uuid.UUID primary key (not
// auto-increment, but still a single PrioritizedPrimaryField), so
// database.PagedQuery applies directly.
func getAllShopsPaged(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db, page)
	}
}

// existsByNpcId returns a provider that checks if a shop exists for a given NPC ID
func existsByNpcId(npcId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		var count int64
		err := db.Model(&Entity{}).
			Where("npc_id = ?", npcId).
			Count(&count).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(count > 0)
	}
}
