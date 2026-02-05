package medal

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity represents a visited map for a medal quest
type Entity struct {
	ID            uint32 `gorm:"primaryKey;autoIncrement;not null"`
	QuestStatusId uint32 `gorm:"not null;index:idx_medal_quest_map,unique"`
	MapId         uint32 `gorm:"not null;index:idx_medal_quest_map,unique"`
}

func (e Entity) TableName() string {
	return "quest_medal_maps"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:    e.ID,
		mapId: _map.Id(e.MapId),
	}, nil
}
