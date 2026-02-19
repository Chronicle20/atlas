package visit

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func getByCharacterIdProvider(characterId uint32) func(db *gorm.DB) model.Provider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("character_id = ?", characterId).Find(&entities)
			return entities, result.Error
		}
	}
}

func getByCharacterIdAndMapIdProvider(characterId uint32) func(mapId _map.Id) func(db *gorm.DB) model.Provider[Entity] {
	return func(mapId _map.Id) func(db *gorm.DB) model.Provider[Entity] {
		return func(db *gorm.DB) model.Provider[Entity] {
			return func() (Entity, error) {
				var entity Entity
				result := db.Where("character_id = ? AND map_id = ?", characterId, uint32(mapId)).First(&entity)
				return entity, result.Error
			}
		}
	}
}
