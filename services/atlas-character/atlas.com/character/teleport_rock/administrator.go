package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]entity, error) {
	var es []entity
	err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
		Order("list_type, slot").
		Find(&es).Error
	return es, err
}

// replaceList rewrites one list wholesale: delete all rows for the
// (character, listType) pair, re-insert the new list with contiguous 0-based
// slots. Lists are at most 10 rows, so full rewrite keeps compaction trivial
// and slot uniqueness conflict-free (design §3).
func replaceList(db *gorm.DB, tenantId uuid.UUID, characterId uint32, listType string, maps []_map.Id) error {
	err := db.Where("tenant_id = ? AND character_id = ? AND list_type = ?", tenantId, characterId, listType).
		Delete(&entity{}).Error
	if err != nil {
		return err
	}
	for i, m := range maps {
		e := &entity{
			ID:          uuid.New(),
			TenantId:    tenantId,
			CharacterId: characterId,
			ListType:    listType,
			Slot:        i,
			MapId:       m,
		}
		if err := db.Create(e).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
		Delete(&entity{}).Error
}

// DeleteForCharacter removes both lists for a character. Called from
// character.Delete's transaction (FR-8 lifecycle cleanup).
func DeleteForCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return deleteByCharacterId(db, tenantId, characterId)
}
