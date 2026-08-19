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

// byCharacterIdPagedProvider pages every ring half a character holds, for
// GET /rings?filter[characterId]=.
func byCharacterIdPagedProvider(t tenant.Model, characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](
			db.Where("tenant_id = ? AND character_id = ?", t.Id(), characterId).Order("created_at, id"), page)
	}
}
