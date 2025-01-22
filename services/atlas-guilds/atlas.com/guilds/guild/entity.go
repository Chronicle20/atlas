package guild

import (
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId            uuid.UUID       `gorm:"not null"`
	Id                  uint32          `gorm:"primaryKey;autoIncrement;not null"`
	WorldId             byte            `gorm:"not null"`
	Name                string          `gorm:"not null"`
	Notice              string          `gorm:"not null"`
	Points              uint32          `gorm:"not null"`
	Capacity            uint32          `gorm:"not null;default=100"`
	Logo                uint16          `gorm:"not null;default=0"`
	LogoColor           byte            `gorm:"not null;default=0"`
	LogoBackground      uint16          `gorm:"not null;default=0"`
	LogoBackgroundColor byte            `gorm:"not null;default=0"`
	AllianceId          uint32          `gorm:"not null;default:0"`
	LeaderId            uint32          `gorm:"not null"`
	Members             []member.Entity `gorm:"foreignkey:GuildId"`
	Titles              []title.Entity  `gorm:"foreignkey:GuildId"`
}

func (e Entity) TableName() string {
	return "guilds"
}

func Make(e Entity) (Model, error) {
	members := make([]member.Model, 0)
	for _, em := range e.Members {
		b, err := member.Make(em)
		if err != nil {
			return Model{}, err
		}
		members = append(members, b)
	}

	titles := make([]title.Model, 0)
	for _, et := range e.Titles {
		b, err := title.Make(et)
		if err != nil {
			return Model{}, err
		}
		titles = append(titles, b)
	}

	return Model{
		tenantId:            e.TenantId,
		id:                  e.Id,
		worldId:             e.WorldId,
		name:                e.Name,
		notice:              e.Notice,
		points:              e.Points,
		capacity:            e.Capacity,
		logo:                e.Logo,
		logoColor:           e.LogoColor,
		logoBackground:      e.LogoBackground,
		logoBackgroundColor: e.LogoBackgroundColor,
		leaderId:            e.LeaderId,
		members:             members,
		titles:              titles,
	}, nil
}
