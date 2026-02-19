package wishlist

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createEntity(db *gorm.DB, t tenant.Model, characterId uint32, serialNumber uint32) (Model, error) {
	e := &Entity{
		TenantId:     t.Id(),
		CharacterId:  characterId,
		SerialNumber: serialNumber,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func deleteEntity(db *gorm.DB, characterId uint32, itemId uuid.UUID) error {
	return db.Where("character_id = ? AND id = ?", characterId, itemId).Delete(&Entity{}).Error
}

func deleteEntityForCharacter(db *gorm.DB, characterId uint32) error {
	return db.Where("character_id = ?", characterId).Delete(&Entity{}).Error
}
