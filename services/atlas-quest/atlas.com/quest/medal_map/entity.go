package medal_map

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
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
