package transaction

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

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
