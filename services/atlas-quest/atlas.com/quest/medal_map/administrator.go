package medal_map

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// recordIfAbsent inserts a (tenant, character, quest, map) row if one does
// not already exist, relying on the unique index rather than a
// read-then-write so the dedup holds under concurrent recordings. Returns
// true when the row was newly created.
func recordIfAbsent(db *gorm.DB, tenantId uuid.UUID, characterId uint32, questId uint32, mapId uint32) (bool, error) {
	m, err := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(characterId).
		SetQuestId(questId).
		SetMapId(_map.Id(mapId)).
		Build()
	if err != nil {
		return false, err
	}

	e := m.ToEntity()
	e.TenantId = tenantId

	res := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}, {Name: "quest_id"}, {Name: "map_id"}},
		DoNothing: true,
	}).Create(&e)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
