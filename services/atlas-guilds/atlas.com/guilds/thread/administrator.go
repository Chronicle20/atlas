package thread

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func create(db *gorm.DB, tenantId uuid.UUID, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
	e := &Entity{
		TenantId:   tenantId,
		GuildId:    guildId,
		PosterId:   posterId,
		Title:      title,
		Message:    message,
		EmoticonId: emoticonId,
		Notice:     notice,
		Replies:    nil,
		CreatedAt:  time.Now(),
	}
	if e.Notice {
		e.Id = 0
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func update(db *gorm.DB, tenantId uuid.UUID, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) error {
	e, err := getById(tenantId, guildId, threadId)(db)()
	if err != nil {
		return err
	}
	e.PosterId = posterId
	e.Title = title
	e.Message = message
	e.EmoticonId = emoticonId
	e.Notice = notice
	err = db.Save(e).Error
	if err != nil {
		return err
	}
	return nil
}

func remove(db *gorm.DB, tenantId uuid.UUID, guildId uint32, threadId uint32) error {
	return db.Where("tenant_id = ? AND guild_id = ? AND id = ?", tenantId, guildId, threadId).Delete(&Entity{}).Error
}
