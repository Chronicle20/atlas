package equipslot

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// activeByCharacterId is the lazy read behind GetActive: the character's
// currently-active extensions, i.e. those whose ExpiresAt is in the future.
// An expired row is not returned and is not deleted -- the history is kept.
func activeByCharacterId(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db.Where("tenant_id = ? AND character_id = ? AND expires_at > ?", tenantId, characterId, time.Now()), &Entity{})
	}
}

// GetActive returns the character's currently-active extensions.
func GetActive(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	return model.SliceMap(Make)(activeByCharacterId(tenantId, characterId)(db))(model.ParallelMap())()
}
