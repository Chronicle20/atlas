package wish

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

func getById(id string) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{Id: parseId(id)})
	}
}

// getBySerial resolves a wish entry by its per-(tenant, world) ITC serial (the
// client's nITCSN). The WHERE is an explicit name-keyed map, not a struct
// condition: GORM elides zero-valued struct fields, which would silently drop
// the world_id filter for world 0 (a valid world.Id, since world.Id is a byte).
// tenant scoping is applied by the tenant query callback from the db's context.
func getBySerial(worldId world.Id, sn uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		var result entity
		err := db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"serial":   sn,
		}).First(&result).Error
		if err != nil {
			return model.ErrorProvider[entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getByCharacterItem resolves the single wish entry for (world, character, item)
// — the uniqueness key the idempotent create enforces. It backs the create-time
// existence check. A missing row yields gorm.ErrRecordNotFound (via First), which
// CreateWish treats as "no existing wish" rather than an error.
func getByCharacterItem(worldId world.Id, characterId uint32, itemId uint32, wishType string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		var result entity
		err := db.Where(map[string]interface{}{
			"world_id":     byte(worldId),
			"character_id": characterId,
			"item_id":      itemId,
			"type":         wishType,
		}).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
		return model.Map(modelFromEntity)(model.FixedProvider(result))
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

// getWantedByWorld returns ALL want-ad entries in a world, across every
// character — the cross-character Wanted tab (ITC_OPERATION section 2). The WHERE
// is an explicit name-keyed map rather than a struct condition: GORM elides
// zero-valued struct fields, which would silently drop the world_id filter for
// world 0 (a valid world.Id, since world.Id is a byte). tenant scoping is applied
// by the tenant query callback from the db's context.
func getWantedByWorld(worldId world.Id) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"type":     TypeWanted,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByCharacterAndType returns a character's wish entries of a single type
// (cart or wanted), so the Cart and Wanted views stay disjoint.
func getByCharacterAndType(characterId uint32, wishType string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"character_id": characterId,
			"type":         wishType,
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
		SetWorldId(world.Id(e.WorldId)).
		SetSerial(e.Serial).
		SetListingSerial(e.ListingSerial).
		SetType(e.Type).
		SetPrice(e.Price).
		SetCount(e.Count).
		SetExpiresAt(e.ExpiresAt).
		SetCreatedAt(e.CreatedAt).
		Build()
}
