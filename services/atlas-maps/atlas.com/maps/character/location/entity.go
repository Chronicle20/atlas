package location

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId    uuid.UUID  `gorm:"type:uuid;primaryKey;not null"`
	CharacterId uint32     `gorm:"primaryKey;not null"`
	WorldId     world.Id   `gorm:"not null"`
	ChannelId   channel.Id `gorm:"not null"`
	MapId       _map.Id    `gorm:"not null"`
	Instance    uuid.UUID  `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000000'"`
	UpdatedAt   time.Time  `gorm:"not null"`
}

func (e entity) TableName() string {
	return "character_locations"
}
