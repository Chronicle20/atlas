package buddy

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id            uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	ListId        uuid.UUID `gorm:"not null"`
	CharacterId   uint32    `gorm:"not null"`
	Group         string    `gorm:"not null"`
	CharacterName string    `gorm:"not null"`
	ChannelId     byte      `gorm:"not null"`
	Visible       bool      `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "buddies"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:            e.Id,
		listId:        e.ListId,
		characterId:   e.CharacterId,
		group:         e.Group,
		characterName: e.CharacterName,
		channelId:     e.ChannelId,
		visible:       e.Visible,
	}, nil
}
