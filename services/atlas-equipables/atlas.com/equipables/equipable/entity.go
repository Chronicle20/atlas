package equipable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId       uuid.UUID `gorm:"not null"`
	ID             uint32    `gorm:"primaryKey;autoIncrement;not null"`
	ItemId         uint32    `gorm:"not null;default=0"`
	Strength       uint16    `gorm:"not null;default=0"`
	Dexterity      uint16    `gorm:"not null;default=0"`
	Intelligence   uint16    `gorm:"not null;default=0"`
	Luck           uint16    `gorm:"not null;default=0"`
	Hp             uint16    `gorm:"not null;default=0"`
	Mp             uint16    `gorm:"not null;default=0"`
	WeaponAttack   uint16    `gorm:"not null;default=0"`
	MagicAttack    uint16    `gorm:"not null;default=0"`
	WeaponDefense  uint16    `gorm:"not null;default=0"`
	MagicDefense   uint16    `gorm:"not null;default=0"`
	Accuracy       uint16    `gorm:"not null;default=0"`
	Avoidability   uint16    `gorm:"not null;default=0"`
	Hands          uint16    `gorm:"not null;default=0"`
	Speed          uint16    `gorm:"not null;default=0"`
	Jump           uint16    `gorm:"not null;default=0"`
	Slots          uint16    `gorm:"not null;default=0"`
	OwnerName      string    `gorm:"not null;default="`
	Locked         bool      `gorm:"not null;default=false"`
	Spikes         bool      `gorm:"not null;default=false"`
	KarmaUsed      bool      `gorm:"not null;default=false"`
	Cold           bool      `gorm:"not null;default=false"`
	CanBeTraded    bool      `gorm:"not null;default=false"`
	LevelType      byte      `gorm:"not null;default=0"`
	Level          byte      `gorm:"not null;default=0"`
	Experience     uint32    `gorm:"not null;default=0"`
	HammersApplied uint32    `gorm:"not null;default=0"`
	Expiration     time.Time `gorm:"not null;default=0"`
}

func (e entity) TableName() string {
	return "equipment"
}
