package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:1"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:2"`
	ListType    string    `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:3"` // "regular" | "vip"
	Slot        int       `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:4"` // 0-based position
	MapId       _map.Id   `gorm:"not null"`
}

func (e entity) TableName() string {
	return "teleport_rock_maps"
}
