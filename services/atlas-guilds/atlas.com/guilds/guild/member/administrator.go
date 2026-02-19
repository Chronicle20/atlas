package member

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error) {
	e := &Entity{
		TenantId:    tenantId,
		GuildId:     guildId,
		CharacterId: characterId,
		Name:        name,
		JobId:       jobId,
		Level:       level,
		Title:       title,
		Online:      true,
	}
	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateStatus(db *gorm.DB, characterId uint32, online bool) error {
	return db.Model(&Entity{}).
		Where("character_id = ?", characterId).
		Update("online", online).Error
}

func updateTitle(db *gorm.DB, characterId uint32, title byte) error {
	return db.Model(&Entity{}).
		Where("character_id = ?", characterId).
		Update("title", title).Error
}
