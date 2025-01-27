package thread

import (
	"atlas-guilds/thread/reply"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId   uuid.UUID      `gorm:"not null"`
	GuildId    uint32         `gorm:"not null"`
	Id         uint32         `gorm:"primaryKey;autoIncrement;not null"`
	PosterId   uint32         `gorm:"not null"`
	Title      string         `gorm:"not null"`
	Message    string         `gorm:"not null"`
	EmoticonId uint32         `gorm:"not null"`
	Notice     bool           `gorm:"not null"`
	Replies    []reply.Entity `gorm:"foreignkey:ThreadId"`
	CreatedAt  time.Time      `gorm:"not null"`
}

func (e *Entity) AfterCreate(tx *gorm.DB) (err error) {
	if e.Notice {
		tx.Model(e).Update("id", "0")
	}
	return
}

func (e Entity) TableName() string {
	return "threads"
}

func Make(e Entity) (Model, error) {
	rs, err := model.SliceMap(reply.Make)(model.FixedProvider(e.Replies))()()
	if err != nil {
		return Model{}, err
	}

	return Model{
		tenantId:   e.TenantId,
		guildId:    e.GuildId,
		id:         e.Id,
		posterId:   e.PosterId,
		title:      e.Title,
		message:    e.Message,
		emoticonId: e.EmoticonId,
		notice:     e.Notice,
		replies:    rs,
		createdAt:  e.CreatedAt,
	}, nil
}
