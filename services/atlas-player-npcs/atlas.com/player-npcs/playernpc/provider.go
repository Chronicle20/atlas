package playernpc

import (
	"gorm.io/gorm"

	"github.com/google/uuid"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// getByIdEntityProvider fetches the root row by id.
func getByIdEntityProvider(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var entity Entity
		err := db.Where("id = ?", id).First(&entity).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(entity)
	}
}

// getEquipmentByPlayerNpcIdProvider fetches every equipment row belonging
// to playerNpcId.
func getEquipmentByPlayerNpcIdProvider(playerNpcId uuid.UUID) database.EntityProvider[[]EquipmentEntity] {
	return func(db *gorm.DB) model.Provider[[]EquipmentEntity] {
		var entities []EquipmentEntity
		err := db.Where("player_npc_id = ?", playerNpcId).Find(&entities).Error
		if err != nil {
			return model.ErrorProvider[[]EquipmentEntity](err)
		}
		return model.FixedProvider(entities)
	}
}

// getByMapPagedProvider fetches the deployed Player NPCs for (world_id,
// map_id) -- the map-enter read path (design §6, PRD §6 index).
func getByMapPagedProvider(worldId byte, mapId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("world_id = ? AND map_id = ?", worldId, mapId), page)
	}
}
