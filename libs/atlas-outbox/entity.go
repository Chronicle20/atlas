package outbox

import (
	"time"

	"gorm.io/datatypes"
)

type Entity struct {
	ID           uint64         `gorm:"primaryKey;column:id"`
	Topic        string         `gorm:"column:topic;not null;index:outbox_entries_unsent_idx,where:sent_at IS NULL"`
	MessageKey   []byte         `gorm:"column:message_key;not null"`
	MessageValue []byte         `gorm:"column:message_value"`
	Headers      datatypes.JSON `gorm:"column:headers;not null;default:'{}'"`
	EnqueuedAt   time.Time      `gorm:"column:enqueued_at;not null;default:CURRENT_TIMESTAMP"`
	SentAt       *time.Time     `gorm:"column:sent_at;index:outbox_entries_sweeper_idx,where:sent_at IS NOT NULL"`
	Attempts     int            `gorm:"column:attempts;not null;default:0"`
	LastError    *string        `gorm:"column:last_error"`
}

func (Entity) TableName() string { return "outbox_entries" }
