package buddy

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	CharacterId   uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	ListId        uuid.UUID `gorm:"not null"`
	Group         string    `gorm:"not null"`
	CharacterName string    `gorm:"not null"`
	ChannelId     int8      `gorm:"not null"`
	Pending       bool      `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "buddies"
}

func Make(e Entity) (Model, error) {
	return Model{
		listId:        e.ListId,
		characterId:   e.CharacterId,
		group:         e.Group,
		characterName: e.CharacterName,
		channelId:     e.ChannelId,
		pending:       e.Pending,
	}, nil
}
