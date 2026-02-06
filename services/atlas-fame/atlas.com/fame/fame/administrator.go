package fame

import (
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

func deleteByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where("tenant_id = ? AND (character_id = ? OR target_id = ?)", tenantId, characterId, characterId).Delete(&Entity{}).Error
}
