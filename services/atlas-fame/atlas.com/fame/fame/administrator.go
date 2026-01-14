package fame

import (
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
	"time"
)

func create(db *gorm.DB, t tenant.Model, characterId uint32, targetId uint32, amount int8) (Model, error) {
	_, err := NewBuilder(t.Id(), characterId, targetId, amount).Build()
	if err != nil {
		return Model{}, err
	}

	e := &Entity{
		TenantId:    t.Id(),
		CharacterId: characterId,
		TargetId:    targetId,
		Amount:      amount,
		CreatedAt:   time.Now(),
	}

	err = db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}
