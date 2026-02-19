package reply

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, threadId uint32, posterId uint32, message string) (Model, error) {
	e := &Entity{
		TenantId:  tenantId,
		ThreadId:  threadId,
		PosterId:  posterId,
		Message:   message,
		CreatedAt: time.Now(),
	}
	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func remove(db *gorm.DB, threadId uint32, replyId uint32) error {
	return db.Where("thread_id = ? AND id = ?", threadId, replyId).Delete(&Entity{}).Error
}
