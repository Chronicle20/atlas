package macro

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"primaryKey;not null"`
	Id          uint32    `gorm:"primaryKey;not null;<-:create;autoIncrement:false"`
	Name        string    `gorm:"not null"`
	Shout       bool      `gorm:"not null"`
	SkillId1    uint32    `gorm:"not null"`
	SkillId2    uint32    `gorm:"not null"`
	SkillId3    uint32    `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "macros"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:       e.Id,
		name:     e.Name,
		shout:    e.Shout,
		skillId1: skill.Id(e.SkillId1),
		skillId2: skill.Id(e.SkillId2),
		skillId3: skill.Id(e.SkillId3),
	}, nil
}
