package member

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId      uuid.UUID `gorm:"not null"`
	CharacterId   uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	GuildId       uint32    `gorm:"not null"`
	Name          string    `gorm:"not null"`
	JobId         uint16    `gorm:"not null;default=0"`
	Level         byte      `gorm:"not null"`
	Title         byte      `gorm:"not null;default=5"`
	Online        bool      `gorm:"not null;default=false"`
	AllianceTitle byte      `gorm:"not null;default=5"`
}

func (e Entity) TableName() string {
	return "members"
}

func Make(e Entity) (Model, error) {
	return Model{
		tenantId:      e.TenantId,
		characterId:   e.CharacterId,
		guildId:       e.GuildId,
		name:          e.Name,
		jobId:         e.JobId,
		level:         e.Level,
		title:         e.Title,
		online:        e.Online,
		allianceTitle: e.AllianceTitle,
	}, nil
}
