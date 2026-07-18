package transaction

import (
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// getByCharacter returns all transaction-history rows owned by a character,
// ordered newest-first (the My Page -> History list).
//
// The filter is an explicit name-keyed map rather than a struct condition:
// GORM's struct-condition Where elides zero-valued fields, so a struct
// condition would silently drop the character_id filter for character 0. The
// map form forces the column into the WHERE clause.
func getByCharacter(characterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"character_id": characterId,
		}).Order("created_at DESC").Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByCharacterPaged backs the REST list handler (GET
// /characters/{characterId}/mts/transactions, task-117), mirroring
// getByCharacter's newest-first ordering. database.PagedQuery appends the
// entity's primary-key ordering as a tie-break AFTER this caller-supplied
// created_at ordering, so pages form a total order even when rows share a
// timestamp.
func getByCharacterPaged(characterId uint32, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(map[string]interface{}{
			"character_id": characterId,
		}).Order("created_at DESC"), page)
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, world.Id(e.WorldId), e.CharacterId).
		SetId(e.Id).
		SetCounterpartyId(e.CounterpartyId).
		SetItemId(e.ItemId).
		SetQuantity(e.Quantity).
		SetTotalPrice(e.TotalPrice).
		SetKind(e.Kind).
		SetCreatedAt(e.CreatedAt).
		Build()
}
