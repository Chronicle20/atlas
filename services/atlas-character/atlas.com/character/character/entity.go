package character

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId           uuid.UUID `gorm:"not null"`
	ID                 uint32    `gorm:"primaryKey;autoIncrement;not null"`
	AccountId          uint32    `gorm:"not null"`
	World              byte      `gorm:"not null"`
	Name               string    `gorm:"not null"`
	Level              byte      `gorm:"not null;default=1"`
	Experience         uint32    `gorm:"not null;default=0"`
	GachaponExperience uint32    `gorm:"not null;default=0"`
	Strength           uint16    `gorm:"not null;default=12"`
	Dexterity          uint16    `gorm:"not null;default=5"`
	Intelligence       uint16    `gorm:"not null;default=4"`
	Luck               uint16    `gorm:"not null;default=4"`
	HP                 uint16    `gorm:"not null;default=50"`
	MP                 uint16    `gorm:"not null;default=5"`
	MaxHP              uint16    `gorm:"not null;default=50"`
	MaxMP              uint16    `gorm:"not null;default=5"`
	Meso               uint32    `gorm:"not null;default=0"`
	HPMPUsed           int       `gorm:"not null;default=0"`
	JobId              uint16    `gorm:"not null;default=0"`
	SkinColor          byte      `gorm:"not null;default=0"`
	Gender             byte      `gorm:"not null;default=0"`
	Fame               int16     `gorm:"not null;default=0"`
	Hair               uint32    `gorm:"not null;default=0"`
	Face               uint32    `gorm:"not null;default=0"`
	AP                 uint16    `gorm:"not null;default=0"`
	SP                 string    `gorm:"not null;default=0,0,0,0,0,0,0,0,0,0"`
	MapId              uint32    `gorm:"not null;default=0"`
	SpawnPoint         uint32    `gorm:"not null;default=0"`
	GM                 int       `gorm:"not null;default=0"`
	X                  int16     `gorm:"not null;default=0"`
	Y                  int16     `gorm:"not null;default=0"`
	Stance             byte      `gorm:"not null;default=0"`
}

func (e entity) TableName() string {
	return "characters"
}
