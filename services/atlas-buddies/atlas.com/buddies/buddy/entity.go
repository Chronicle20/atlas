package buddy

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&Entity{}); err != nil {
		return err
	}
	if !db.Migrator().HasTable("lists") {
		return nil
	}
	return db.Exec(`
		UPDATE buddies
		SET tenant_id = (SELECT tenant_id FROM lists WHERE lists.id = buddies.list_id)
		WHERE (tenant_id IS NULL OR tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND EXISTS (SELECT 1 FROM lists WHERE lists.id = buddies.list_id)
	`).Error
}

type Entity struct {
	CharacterId   uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	ListId        uuid.UUID `gorm:"not null"`
	TenantId      uuid.UUID `gorm:"not null;index"`
	Group         string    `gorm:"not null"`
	CharacterName string    `gorm:"not null"`
	ChannelId     int8      `gorm:"not null;default:-1"`
	InShop        bool      `gorm:"not null;default:false"`
	Pending       bool      `gorm:"not null;default:false"`
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
		inShop:        e.InShop,
		pending:       e.Pending,
	}, nil
}
