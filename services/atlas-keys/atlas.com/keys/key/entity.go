package key

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	Key         int32     `gorm:"primaryKey;autoIncrement:false;not null"`
	Type        int8      `gorm:"not null"`
	Action      int32     `gorm:"not null"`
}

func (e entity) TableName() string {
	return "keys"
}

// Make transforms an entity into a Model.
func Make(e entity) (Model, error) {
	return Model{
		characterId: e.CharacterId,
		key:         e.Key,
		theType:     e.Type,
		action:      e.Action,
	}, nil
}

// ToEntity transforms a Model into an entity for persistence.
func (m Model) ToEntity(tenantId uuid.UUID) entity {
	return entity{
		TenantId:    tenantId,
		CharacterId: m.characterId,
		Key:         m.key,
		Type:        m.theType,
		Action:      m.action,
	}
}
