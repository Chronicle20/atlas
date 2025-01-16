package list

import (
	"atlas-buddies/buddy"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId    uuid.UUID      `gorm:"not null"`
	Id          uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()"`
	CharacterId uint32         `gorm:"not null"`
	Capacity    uint32         `gorm:"not null"`
	Buddies     []buddy.Entity `gorm:"foreignkey:ListId"`
}

func (e Entity) TableName() string {
	return "lists"
}

func Make(e Entity) (Model, error) {
	buddies := make([]buddy.Model, 0)
	for _, eb := range e.Buddies {
		b, err := buddy.Make(eb)
		if err != nil {
			return Model{}, err
		}
		buddies = append(buddies, b)
	}

	return Model{
		tenantId:    e.TenantId,
		id:          e.Id,
		characterId: e.CharacterId,
		capacity:    e.Capacity,
		buddies:     buddies,
	}, nil
}
