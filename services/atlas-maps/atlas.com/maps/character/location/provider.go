package location

import (
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getByTenantAndCharacterIdProvider returns the persistence-layer Provider for
// the row keyed by (tenantId, characterId). It mirrors the visit/ peer's
// curried provider shape: characterId -> tenantId -> db -> Provider[entity].
func getByTenantAndCharacterIdProvider(characterId uint32) func(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[entity] {
	return func(tenantId uuid.UUID) func(db *gorm.DB) model.Provider[entity] {
		return func(db *gorm.DB) model.Provider[entity] {
			return func() (entity, error) {
				var e entity
				result := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).First(&e)
				return e, result.Error
			}
		}
	}
}
