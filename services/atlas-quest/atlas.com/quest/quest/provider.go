package quest

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func byIdEntityProvider(id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).Preload("Progress").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// byCharacterIdPagedEntityProvider backs the REST list handler (GET
// /characters/{characterId}/quests, task-117). The prior unpaged
// byCharacterIdEntityProvider had no internal caller besides that handler,
// so ByCharacterIdProvider/GetByCharacterId were converted in place rather
// than kept alongside this as a separate unpaged method.
func byCharacterIdPagedEntityProvider(characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("character_id = ?", characterId).Preload("Progress"), page)
	}
}

func byCharacterIdAndQuestIdEntityProvider(characterId uint32, questId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("character_id = ? AND quest_id = ?", characterId, questId).Preload("Progress").First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

func byCharacterIdAndStateEntityProvider(characterId uint32, state State) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("character_id = ? AND state = ?", characterId, state).Preload("Progress").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// byCharacterIdAndStatePagedEntityProvider backs the REST list handlers
// only (GET /characters/{characterId}/quests/started|completed, task-117).
// byCharacterIdAndStateEntityProvider above stays unpaged: the atlas-quest
// monster-kill and character Kafka consumers scan EVERY started quest for a
// character on every monster kill / character event (a hot game path) and
// must see the complete set, not one page.
func byCharacterIdAndStatePagedEntityProvider(characterId uint32, state State, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("character_id = ? AND state = ?", characterId, state).Preload("Progress"), page)
	}
}
