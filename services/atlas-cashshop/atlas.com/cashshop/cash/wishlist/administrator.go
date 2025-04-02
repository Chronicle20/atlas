package wishlist

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func deleteEntity(ctx context.Context) func(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return func(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
		return db.WithContext(ctx).Where("character_id = ?", characterId).Delete(&Entity{}).Error
	}
}
