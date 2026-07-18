package holding

import (
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// getAll resolves every holding row visible to the request's tenant. It backs
// ONLY the administrator.GetAll() test-verification wrapper (full-table DB
// assertions in test code) — the REST-facing GetAll() was removed from
// Processor in favor of the paged ByOwnerPagedProvider/ByCharacterPagedProvider
// (task-117); this unfiltered provider is never reachable from a handler.
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

// getBySerial resolves a holding by its per-(tenant, world) ITC serial (the
// client's nITCSN). The WHERE clause is an explicit name-keyed map rather than a
// struct condition: GORM's struct-condition Where elides zero-valued fields, so a
// struct condition would silently drop the world_id filter for world 0 (a valid
// world.Id, since world.Id is a byte) and resolve the wrong row. tenant scoping is
// applied by the tenant query callback from the db's context; the default (non
// soft-deleted) scope applies, so a taken-home holding is not resolved.
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

// getByOwner returns the holdings for a character in a world.
//
// The filter is built as an explicit name-keyed map rather than a struct
// condition: GORM's struct-condition Where elides zero-valued fields, so a
// struct condition would silently drop the world_id filter for world 0 (a
// valid world.Id, since world.Id is a byte) and return cross-world rows. The
// map form forces every filter column into the WHERE clause.
func getByOwner(worldId world.Id, ownerId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"owner_id": ownerId,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByCharacter returns all holdings for a character (owner) across worlds.
//
// The route is keyed on characterId alone, so this provider does not constrain
// world_id. The owner_id filter uses an explicit name-keyed map (never a struct
// condition) so owner 0 is not silently dropped.
func getByCharacter(ownerId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"owner_id": ownerId,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByOwnerPaged backs the REST list handler's ?worldId= narrowed branch (GET
// /characters/{characterId}/mts/holding?worldId=, task-117). Single-PK entity
// (surrogate UUID), so this is a straight database.PagedQuery over the same
// world_id+owner_id scope getByOwner uses.
func getByOwnerPaged(worldId world.Id, ownerId uint32, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"owner_id": ownerId,
		}), page)
	}
}

// getByCharacterPaged backs the REST list handler's unscoped branch (GET
// /characters/{characterId}/mts/holding, task-117), mirroring getByCharacter.
func getByCharacterPaged(ownerId uint32, page model.Page) database.EntityProvider[model.Paged[entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[entity]] {
		return database.PagedQuery[entity](db.Where(map[string]interface{}{
			"owner_id": ownerId,
		}), page)
	}
}

func modelFromEntity(e entity) (Model, error) {
	b := NewBuilder(e.TenantId, world.Id(e.WorldId), e.OwnerId).
		SetId(e.Id).
		SetSerial(e.Serial).
		SetOrigin(Origin(e.Origin)).
		SetTemplateId(e.TemplateId).
		SetQuantity(e.Quantity).
		SetStrength(e.Strength).
		SetDexterity(e.Dexterity).
		SetIntelligence(e.Intelligence).
		SetLuck(e.Luck).
		SetHP(e.HP).
		SetMP(e.MP).
		SetWeaponAttack(e.WeaponAttack).
		SetMagicAttack(e.MagicAttack).
		SetWeaponDefense(e.WeaponDefense).
		SetMagicDefense(e.MagicDefense).
		SetAccuracy(e.Accuracy).
		SetAvoidability(e.Avoidability).
		SetHands(e.Hands).
		SetSpeed(e.Speed).
		SetJump(e.Jump).
		SetSlots(e.Slots).
		SetLevel(e.Level).
		SetItemLevel(e.ItemLevel).
		SetItemExp(e.ItemExp).
		SetRingId(e.RingId).
		SetViciousCount(e.ViciousCount).
		SetFlags(e.Flags).
		SetOwner(e.Owner).
		SetCreatedAt(e.CreatedAt)
	return b.Build()
}
