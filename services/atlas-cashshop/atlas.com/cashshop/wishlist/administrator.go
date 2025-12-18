package wishlist

import (
	"context"
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

func deleteEntity(ctx context.Context) func(db *gorm.DB, tenantId uuid.UUID, characterId uint32, itemId uuid.UUID) error {
	return func(db *gorm.DB, tenantId uuid.UUID, characterId uint32, itemId uuid.UUID) error {
		return db.WithContext(ctx).Where("tenant_id = ? AND character_id = ? AND id = ?", tenantId, characterId, itemId).Delete(&Entity{}).Error
	}
}

func deleteEntityForCharacter(ctx context.Context) func(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return func(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
		return db.WithContext(ctx).Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Delete(&Entity{}).Error
	}
}
