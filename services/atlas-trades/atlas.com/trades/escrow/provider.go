package escrow

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ItemsByRoom yields one room's live escrowed items, oldest first, scoped to the
// request tenant. Trade-slot order would be the display order, but the unwind
// and the settlement both care about determinism rather than presentation, and
// creation order is stable even when two slots were filled in one tick.
func ItemsByRoom(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID) ([]ItemModel, error) {
	return func(roomId uuid.UUID) ([]ItemModel, error) {
		var entities []ItemEntity
		err := db.
			Where("tenant_id = ? AND room_id = ?", tenantId, roomId).
			Order("created_at ASC, id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}
		return model.SliceMap(MakeItem)(model.FixedProvider(entities))(model.ParallelMap())()
	}
}

// ItemById yields one escrowed item by row id, scoped to the request tenant.
func ItemById(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) (ItemModel, error) {
	return func(id uuid.UUID) (ItemModel, error) {
		var e ItemEntity
		if err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&e).Error; err != nil {
			return ItemModel{}, err
		}
		return MakeItem(e)
	}
}

// MesosByRoom yields one room's escrowed meso rows, scoped to the request
// tenant. At most one row per participant.
func MesosByRoom(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID) ([]MesoModel, error) {
	return func(roomId uuid.UUID) ([]MesoModel, error) {
		var entities []MesoEntity
		err := db.
			Where("tenant_id = ? AND room_id = ?", tenantId, roomId).
			Order("owner_id ASC").
			Find(&entities).Error
		if err != nil {
			return nil, err
		}
		return model.SliceMap(MakeMeso)(model.FixedProvider(entities))(model.ParallelMap())()
	}
}

// MesoByOwner yields one participant's escrowed meso total for a room, and
// whether a row existed at all. A missing row means zero escrowed, which is a
// legitimate state, so absence is NOT an error — the delta arithmetic in
// staging treats it as 0.
func MesoByOwner(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) (int64, bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id) (int64, bool, error) {
		var e MesoEntity
		err := db.Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).First(&e).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return 0, false, nil
			}
			return 0, false, err
		}
		return e.Amount, true, nil
	}
}

// MesoStakeById finds one in-flight stake by the id its saga was submitted
// with.
//
// Deliberately NOT tenant-scoped, matching AllItems/AllMesos: the caller
// resolving an award_mesos saga's terminal status has only the stakeId it
// submitted the saga with — by the time the status arrives the room that knew
// the tenant may already be gone (that is the whole reason this durable path
// exists, see MesoStakeEntity's doc comment). The row itself carries the tenant
// quad needed to rebuild the tenant once found (MesoStakeModel.Tenant).
func MesoStakeById(db *gorm.DB) func(stakeId uuid.UUID) (MesoStakeModel, bool, error) {
	return func(stakeId uuid.UUID) (MesoStakeModel, bool, error) {
		var e MesoStakeEntity
		err := db.Where("id = ?", stakeId).First(&e).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return MesoStakeModel{}, false, nil
			}
			return MesoStakeModel{}, false, err
		}
		m, err := MakeMesoStake(e)
		if err != nil {
			return MesoStakeModel{}, false, err
		}
		return m, true, nil
	}
}

// EffectiveMesoByOwner is the total a participant currently has STAKED: what is
// committed plus every delta still in flight. It is the figure a new stage must
// net against, and the item column's StagedQuantityFrom is its twin — both
// answer "how much of this have I already moved toward the trade", counting
// movements that have been submitted but not yet confirmed.
//
// Netting against the committed Amount ALONE was the defect this exists to
// close: a player retyping the box before the first saga resolved had the
// second delta computed as if the first had never happened, so both sagas
// debited the full amount and the difference was destroyed.
//
// Signed, because a reduction in flight makes the sum legitimately negative
// while the settled figure never is.
func EffectiveMesoByOwner(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) (int64, error) {
	return func(roomId uuid.UUID, ownerId character.Id) (int64, error) {
		committed, _, err := MesoByOwner(db, tenantId)(roomId, ownerId)
		if err != nil {
			return 0, err
		}
		inFlight, err := InFlightMesoDelta(db, tenantId)(roomId, ownerId)
		if err != nil {
			return 0, err
		}
		return committed + inFlight, nil
	}
}

// InFlightMesoDelta sums the signed movements of every stake still outstanding
// for a participant. Zero when none are.
func InFlightMesoDelta(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) (int64, error) {
	return func(roomId uuid.UUID, ownerId character.Id) (int64, error) {
		// COALESCE because SUM over no rows is NULL, which will not scan into
		// an int64.
		var total int64
		err := db.Model(&MesoStakeEntity{}).
			Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
			Select("COALESCE(SUM(delta), 0)").
			Scan(&total).Error
		return total, err
	}
}

// MesoStakesByOwner yields every stake still outstanding for a participant,
// oldest first.
func MesoStakesByOwner(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) ([]MesoStakeModel, error) {
	return func(roomId uuid.UUID, ownerId character.Id) ([]MesoStakeModel, error) {
		var entities []MesoStakeEntity
		if err := db.Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
			Order("created_at ASC, id ASC").Find(&entities).Error; err != nil {
			return nil, err
		}
		return model.SliceMap(MakeMesoStake)(model.FixedProvider(entities))(model.ParallelMap())()
	}
}

// AllMesoStakes yields EVERY outstanding stake across EVERY tenant, oldest
// first — the stake twin of AllMesos, and un-scoped for the same reason.
func AllMesoStakes(db *gorm.DB) ([]MesoStakeModel, error) {
	var entities []MesoStakeEntity
	if err := db.Order("created_at ASC, id ASC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return model.SliceMap(MakeMesoStake)(model.FixedProvider(entities))(model.ParallelMap())()
}

// AllItems yields EVERY live escrowed item across EVERY tenant, oldest first.
//
// Deliberately un-scoped: startup reconciliation runs before any request has
// supplied a tenant, and each row carries the tenant quad needed to restore one
// (see ItemEntity). Rooms are process-local, so every row this returns at boot
// is an orphan by definition and is returned to its owner (design §5A.9).
//
// This and AllMesos are the ONLY queries in the package that cross tenants, and
// both are reachable only from the boot path and the retry ticker.
func AllItems(db *gorm.DB) ([]ItemModel, error) {
	var entities []ItemEntity
	if err := db.Order("created_at ASC, id ASC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return model.SliceMap(MakeItem)(model.FixedProvider(entities))(model.ParallelMap())()
}

// AllMesos yields EVERY escrowed meso row across EVERY tenant. See AllItems.
func AllMesos(db *gorm.DB) ([]MesoModel, error) {
	var entities []MesoEntity
	if err := db.Order("created_at ASC, id ASC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return model.SliceMap(MakeMeso)(model.FixedProvider(entities))(model.ParallelMap())()
}
