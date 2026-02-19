package visit

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func recordVisit(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) func(mapId _map.Id) error {
	return func(tenantId uuid.UUID) func(characterId uint32) func(mapId _map.Id) error {
		return func(characterId uint32) func(mapId _map.Id) error {
			return func(mapId _map.Id) error {
				entity := Entity{
					ID:          uuid.New(),
					TenantID:    tenantId,
					CharacterID: characterId,
					MapID:       uint32(mapId),
				}
				result := db.Where("character_id = ? AND map_id = ?", characterId, uint32(mapId)).FirstOrCreate(&entity)
				return result.Error
			}
		}
	}
}

func deleteByCharacterId(db *gorm.DB) func(characterId uint32) (int64, error) {
	return func(characterId uint32) (int64, error) {
		result := db.Where("character_id = ?", characterId).Delete(&Entity{})
		if result.Error != nil {
			return 0, result.Error
		}
		return result.RowsAffected, nil
	}
}
