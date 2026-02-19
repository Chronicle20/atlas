package key

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, characterId uint32, key int32, theType int8, action int32) (Model, error) {
	e := &entity{
		TenantId:    tenantId,
		CharacterId: characterId,
		Key:         key,
		Type:        theType,
		Action:      action,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func update(db *gorm.DB, characterId uint32, key int32, theType int8, action int32) error {
	return db.Model(&entity{CharacterId: characterId, Key: key}).Select("Type", "Action").Updates(entity{Type: theType, Action: action}).Error
}

func deleteByCharacter(db *gorm.DB, characterId uint32) error {
	return db.Where("character_id = ?", characterId).Delete(&entity{}).Error
}
