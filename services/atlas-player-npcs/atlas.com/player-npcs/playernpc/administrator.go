package playernpc

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// createPlayerNpc inserts the root row and its equipment rows in a single
// transaction (design §8: the whole deploy is one atomic write). A unique
// constraint violation on script id, (map, name), or (map, object id) --
// PRD §6 -- aborts the transaction and is returned to the caller as-is.
func createPlayerNpc(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	entity := MakeEntity(tenantId, m)
	entity.Id = uuid.Nil

	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}

		equipment := MakeEquipmentEntities(tenantId, entity.Id, m)
		for i := range equipment {
			equipment[i].Id = uuid.Nil
			if err := tx.Create(&equipment[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Model{}, err
	}

	return getPlayerNpcModel(db, entity.Id)
}

// deletePlayerNpc removes the root row. The `player_npc_equipment` foreign
// key is declared ON DELETE CASCADE (entity.go), so its rows are removed by
// the database, not by this function.
func deletePlayerNpc(db *gorm.DB, id uuid.UUID) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&Entity{}).Error
	})
}

// getPlayerNpcModel hydrates a full Model (root + equipment) for id.
func getPlayerNpcModel(db *gorm.DB, id uuid.UUID) (Model, error) {
	entity, err := getByIdEntityProvider(id)(db)()
	if err != nil {
		return Model{}, err
	}
	equipmentEntities, err := getEquipmentByPlayerNpcIdProvider(id)(db)()
	if err != nil {
		return Model{}, err
	}
	return Make(entity, equipmentEntities)
}

// playerNpcsByMap hydrates every deployed Player NPC for (worldId, mapId)
// -- the map-enter read path (design §6).
func playerNpcsByMap(db *gorm.DB, worldId byte, mapId uint32, page model.Page) ([]Model, error) {
	paged, err := getByMapPagedProvider(worldId, mapId, page)(db)()
	if err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(paged.Items))
	for _, e := range paged.Items {
		m, err := getPlayerNpcModel(db, e.Id)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}
