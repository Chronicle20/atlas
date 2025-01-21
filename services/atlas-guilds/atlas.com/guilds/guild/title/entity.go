package title

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId uuid.UUID `gorm:"not null"`
	Id       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	GuildId  uint32    `json:"guildId"`
	Name     string    `json:"name"`
	Index    byte      `json:"index"`
}

func (e Entity) TableName() string {
	return "members"
}

func Make(e Entity) (Model, error) {
	return Model{
		tenantId: e.TenantId,
		id:       e.Id,
		guildId:  e.GuildId,
		name:     e.Name,
		index:    e.Index,
	}, nil
}
