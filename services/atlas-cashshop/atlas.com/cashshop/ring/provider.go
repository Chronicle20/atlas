package ring

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func byCharacterIdProvider(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Find(&entities)
			return entities, result.Error
		}
	}
}

// byPairIdProvider returns every row sharing pairId -- normally both halves
// of the pair, occasionally one (a partial pair predating FR-RING-4, or a
// row from another tenant excluded by the tenant_id scope).
func byPairIdProvider(tenantId uuid.UUID, pairId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return func() ([]Entity, error) {
			var entities []Entity
			result := db.Where("tenant_id = ? AND pair_id = ?", tenantId, pairId).Find(&entities)
			return entities, result.Error
		}
	}
}

func byIdProvider(tenantId uuid.UUID, id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity)
			return entity, result.Error
		}
	}
}

// GetByCharacterId returns every ring half the character holds, across both
// pair types and all states. A character with none is (nil-length slice,
// nil error) rather than an error.
func GetByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	return model.SliceMap(Make)(byCharacterIdProvider(tenantId, characterId)(db))(model.ParallelMap())()
}

// GetById returns one ring half by its own id.
func GetById(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) (Model, error) {
	e, err := byIdProvider(tenantId, id)(db)()
	if err != nil {
		return Model{}, err
	}
	return Make(e)
}

// GetByPairId returns every row sharing pairId -- see byPairIdProvider.
func GetByPairId(db *gorm.DB, tenantId uuid.UUID, pairId uuid.UUID) ([]Model, error) {
	return model.SliceMap(Make)(byPairIdProvider(tenantId, pairId)(db))(model.ParallelMap())()
}

// byCharacterIdPagedProvider pages every ring half a character holds, for
// GET /rings?filter[characterId]=.
func byCharacterIdPagedProvider(t tenant.Model, characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](
			db.Where("tenant_id = ? AND character_id = ?", t.Id(), characterId).Order("created_at, id"), page)
	}
}
