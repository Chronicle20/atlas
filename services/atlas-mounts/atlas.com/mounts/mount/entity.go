package mount

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId            uuid.UUID  `gorm:"not null;uniqueIndex:idx_character_mount_lookup,priority:1"`
	CharacterId         uint32     `gorm:"not null;uniqueIndex:idx_character_mount_lookup,priority:2"`
	Id                  uuid.UUID  `gorm:"primary_key"`
	Level               int        `gorm:"not null;default:1"`
	Exp                 int        `gorm:"not null;default:0"`
	Tiredness           int        `gorm:"not null;default:0"`
	LastTirednessTickAt *time.Time `gorm:""`
}

func (e Entity) TableName() string {
	return "character_mounts"
}

func Make(e Entity) (Model, error) {
	return NewModelBuilder(e.TenantId, e.CharacterId, e.Id).
		SetLevel(e.Level).
		SetExp(e.Exp).
		SetTiredness(e.Tiredness).
		SetLastTirednessTickAt(e.LastTirednessTickAt).
		Build()
}
