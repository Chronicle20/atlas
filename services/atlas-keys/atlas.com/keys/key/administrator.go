package key

import (
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func create(db *gorm.DB, t tenant.Model, characterId uint32, key int32, theType int8, action int32) (Model, error) {
	e := &entity{
		TenantId:    t.Id(),
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

func update(db *gorm.DB, t tenant.Model, characterId uint32, key int32, theType int8, action int32) error {
	return db.Model(&entity{TenantId: t.Id(), CharacterId: characterId, Key: key}).Select("Type", "Action").Updates(entity{Type: theType, Action: action}).Error
}

func deleteByCharacter(db *gorm.DB, t tenant.Model, characterId uint32) error {
	return db.Where(&entity{TenantId: t.Id(), CharacterId: characterId}).Delete(&entity{}).Error
}
