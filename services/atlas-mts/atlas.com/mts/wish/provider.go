package wish

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

func getById(id string) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{Id: parseId(id)})
	}
}

// getByCharacter returns the wish entries for a character.
//
// The filter is built as an explicit name-keyed map rather than a struct
// condition: GORM's struct-condition Where elides zero-valued fields, so the
// map form forces the character_id column into the WHERE clause regardless of
// the value.
func getByCharacter(characterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"character_id": characterId,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, e.CharacterId, e.ItemId).
		SetId(e.Id).
		SetCreatedAt(e.CreatedAt).
		Build()
}
