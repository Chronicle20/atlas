package character

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"primaryKey;not null"`
	GuildId     uint32    `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "characters"
}

func Make(e Entity) (Model, error) {
	return Model{
		tenantId:    e.TenantId,
		characterId: e.CharacterId,
		guildId:     e.GuildId,
	}, nil
}
