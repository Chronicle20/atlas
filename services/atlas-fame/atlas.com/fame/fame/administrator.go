package fame

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, characterId uint32, targetId uint32, amount int8) (Model, error) {
	_, err := NewBuilder(tenantId, characterId, targetId, amount).Build()
	if err != nil {
		return Model{}, err
	}

	e := &Entity{
		TenantId:    tenantId,
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

func deleteByCharacterId(db *gorm.DB, characterId uint32) error {
	return db.Where("character_id = ? OR target_id = ?", characterId, characterId).Delete(&Entity{}).Error
}
