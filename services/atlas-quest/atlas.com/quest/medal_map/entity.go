package medal_map

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// entity is one (tenant, character, quest, map) visited-map record. The
// unique index enforces Cosmic's per-quest visited-map dedup
// (qs.addMedalMap, MapScriptMethods.java:104-139) at the database level
// rather than by read-then-write (task-290 G14).
type entity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_medal_map_visit,priority:1"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_medal_map_visit,priority:2"`
	QuestId     uint32    `gorm:"not null;uniqueIndex:idx_medal_map_visit,priority:3"`
	MapId       uint32    `gorm:"not null;uniqueIndex:idx_medal_map_visit,priority:4"`
}

func (e entity) TableName() string {
	return "quest_medal_map_visits"
}

// Make converts a persisted entity into its domain Model.
func Make(e entity) (Model, error) {
	return NewBuilder().
		SetId(e.ID).
		SetCharacterId(e.CharacterId).
		SetQuestId(e.QuestId).
		SetMapId(_map.Id(e.MapId)).
		BuildWithValidation()
}

// ToEntity is the inverse of Make for the fields Model owns. TenantId is not
// carried on Model -- recordIfAbsent (administrator.go) sets it directly
// from the tenant in context when persisting, so it is not part of this
// round-trip.
func (m Model) ToEntity() entity {
	return entity{
		ID:          m.id,
		CharacterId: m.characterId,
		QuestId:     m.questId,
		MapId:       uint32(m.mapId),
	}
}
