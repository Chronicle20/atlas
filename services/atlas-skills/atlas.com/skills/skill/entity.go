package skill

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"not null"`
	Id          uint32    `gorm:"primaryKey;not null"`
	Level       byte      `gorm:"not null"`
	MasterLevel byte      `gorm:"not null"`
	Expiration  time.Time `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "skills"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:          e.Id,
		level:       e.Level,
		masterLevel: e.MasterLevel,
		expiration:  e.Expiration,
	}, nil
}
