package reply

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId  uuid.UUID `gorm:"not null"`
	ThreadId  uint32    `gorm:"not null"`
	Id        uint32    `gorm:"primaryKey;autoIncrement;not null"`
	PosterId  uint32    `gorm:"not null"`
	Message   string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "replies"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:        e.Id,
		posterId:  e.PosterId,
		message:   e.Message,
		createdAt: e.CreatedAt,
	}, nil
}
