package list

import (
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func create(db *gorm.DB, t tenant.Model, characterId uint32, capacity uint32) (Model, error) {
	e := &Entity{
		TenantId:    t.Id(),
		CharacterId: characterId,
		Capacity:    capacity,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}
