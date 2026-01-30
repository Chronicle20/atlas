package history

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement;not null"`
	TenantId    uuid.UUID  `gorm:"not null;index:idx_session_history_lookup,priority:1"`
	CharacterId uint32     `gorm:"not null;index:idx_session_history_lookup,priority:2"`
	WorldId     world.Id   `gorm:"not null"`
	ChannelId   channel.Id `gorm:"not null"`
	LoginTime   time.Time  `gorm:"not null;index:idx_session_history_lookup,priority:3"`
	LogoutTime  *time.Time `gorm:"default:null"`
}

func (e entity) TableName() string {
	return "session_history"
}
