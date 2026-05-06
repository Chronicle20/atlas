package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration runs AutoMigrate for the entity, then explicitly drops the
// legacy MapId/Instance columns. atlas-maps owns character location state
// (task-055); GORM's AutoMigrate adds columns but never removes them, so the
// drop is performed here. Idempotent — safe to re-run.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity{}); err != nil {
		return err
	}
	if db.Migrator().HasColumn(&entity{}, "MapId") {
		if err := db.Migrator().DropColumn(&entity{}, "MapId"); err != nil {
			return err
		}
	}
	if db.Migrator().HasColumn(&entity{}, "Instance") {
		if err := db.Migrator().DropColumn(&entity{}, "Instance"); err != nil {
			return err
		}
	}
	return nil
}

type entity struct {
	TenantId           uuid.UUID `gorm:"not null"`
	ID                 uint32    `gorm:"primaryKey;autoIncrement;not null"`
	AccountId          uint32    `gorm:"not null"`
	World              world.Id  `gorm:"not null"`
	Name               string    `gorm:"not null"`
	Level              byte      `gorm:"not null;default=1"`
	Experience         uint32    `gorm:"not null;default=0"`
	GachaponExperience uint32    `gorm:"not null;default=0"`
	Strength           uint16    `gorm:"not null;default=12"`
	Dexterity          uint16    `gorm:"not null;default=5"`
	Intelligence       uint16    `gorm:"not null;default=4"`
	Luck               uint16    `gorm:"not null;default=4"`
	Hp                 uint16    `gorm:"not null;default=50"`
	Mp                 uint16    `gorm:"not null;default=5"`
	MaxHp              uint16    `gorm:"not null;default=50"`
	MaxMp              uint16    `gorm:"not null;default=5"`
	Meso               uint32    `gorm:"not null;default=0"`
	HpMpUsed           int       `gorm:"column:hpmp_used;not null;default=0"`
	JobId              job.Id    `gorm:"not null;default=0"`
	SkinColor          byte      `gorm:"not null;default=0"`
	Gender             byte      `gorm:"not null;default=0"`
	Fame               int16     `gorm:"not null;default=0"`
	Hair               uint32    `gorm:"not null;default=0"`
	Face               uint32    `gorm:"not null;default=0"`
	AP                 uint16    `gorm:"not null;default=0"`
	SP                 string    `gorm:"not null;default=0,0,0,0,0,0,0,0,0,0"`
	SpawnPoint         uint32    `gorm:"not null;default=0"`
	GM                 int       `gorm:"not null;default=0"`
	X                  int16     `gorm:"not null;default=0"`
	Y                  int16     `gorm:"not null;default=0"`
	Stance             byte      `gorm:"not null;default=0"`
}

func (e entity) TableName() string {
	return "characters"
}
