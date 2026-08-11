package escrow

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// toItemEntity maps the immutable model onto its row, stamping the tenant quad
// so the row can rebuild its own tenant during startup reconciliation.
//
// The snapshot is EXPLODED into name-keyed columns rather than stored as a JSON
// blob, matching atlas-mts's holdings table: a binary COPY/restore is then
// column-order safe, and a stat is queryable when a stuck row has to be
// diagnosed by hand.
func toItemEntity(t tenant.Model, m ItemModel) ItemEntity {
	s := m.Snapshot()
	return ItemEntity{
		Id:                  m.Id(),
		TenantId:            t.Id(),
		TenantRegion:        t.Region(),
		TenantMajor:         t.MajorVersion(),
		TenantMinor:         t.MinorVersion(),
		RoomId:              m.RoomId(),
		OwnerId:             m.OwnerId(),
		TradeSlot:           m.TradeSlot(),
		SourceInventoryType: m.SourceInventoryType(),
		SourceSlot:          slot.Position(s.Slot),
		AssetId:             m.AssetId(),
		TemplateId:          item.Id(s.TemplateId),
		Quantity:            asset.Quantity(s.Quantity),
		Expiration:          s.Expiration,
		CashId:              s.CashId,
		Rechargeable:        s.Rechargeable,
		Strength:            s.Strength,
		Dexterity:           s.Dexterity,
		Intelligence:        s.Intelligence,
		Luck:                s.Luck,
		HP:                  s.Hp,
		MP:                  s.Mp,
		WeaponAttack:        s.WeaponAttack,
		MagicAttack:         s.MagicAttack,
		WeaponDefense:       s.WeaponDefense,
		MagicDefense:        s.MagicDefense,
		Accuracy:            s.Accuracy,
		Avoidability:        s.Avoidability,
		Hands:               s.Hands,
		Speed:               s.Speed,
		Jump:                s.Jump,
		Slots:               s.Slots,
		LevelType:           s.LevelType,
		Level:               s.Level,
		Experience:          s.Experience,
		HammersApplied:      s.HammersApplied,
		Flags:               s.Flag,
		Owner:               s.Owner,
		PetId:               s.PetId,
		PetName:             s.PetName,
		PetLevel:            s.PetLevel,
		Closeness:           s.Closeness,
		Fullness:            s.Fullness,
	}
}

// CreateItem writes one escrow row.
//
// The write is idempotent on the row id: a redelivered ACCEPT_TO_TRADE command
// must not create a second row for the same escrow id, because the unwind would
// then return the item twice. Kafka is at-least-once and the custody consumer
// has no other dedupe, so the conflict clause is the guard.
func CreateItem(db *gorm.DB, t tenant.Model) func(m ItemModel) error {
	return func(m ItemModel) error {
		e := toItemEntity(t, m)
		return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(&e).Error
	}
}

// DeleteItem soft-deletes one escrow row.
//
// A delete that matches nothing is SUCCESS, not an error. The unwind retries
// (design §5A.8) and a settlement release can be redelivered; treating "already
// gone" as a failure would fail a saga step whose effect had already landed and
// trigger a compensation that restores the row.
func DeleteItem(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		return db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&ItemEntity{}).Error
	}
}

// ClaimItemForReturn claims the exclusive right to submit a trade_unwind for one
// escrow row, and reports whether THIS caller won it.
//
// The compare-and-set happens INSIDE the UPDATE's WHERE clause — decided by
// RowsAffected, never by a read followed by a write — for the same reason
// CommitMesoStake works that way: the two callers that can each decide to return
// a row (a room teardown reading ItemsByRoom, and an orphaned stage's terminal
// status reading ItemById) run in different transactions with no ordering
// between them, so a read-then-write would let both observe "unclaimed" and both
// submit. The item would then be granted to its owner twice — the meso twin has
// never had that bug precisely because its stake is consumed by the statement
// that acts on it.
//
// The claim is a COLUMN, so it survives a restart: the boot sweep re-reads every
// surviving row, and a row whose unwind is already in flight must not be swept
// into a second one. Claim and submission commit together (the caller runs both
// inside emit's transaction with the outbox), so a crash before the commit
// leaves the row unclaimed and fully retryable.
//
// A row that is already soft-deleted cannot be claimed: the default scope
// excludes it, which is correct — its item has left custody and there is nothing
// left to return.
func ClaimItemForReturn(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) (bool, error) {
	return func(id uuid.UUID) (bool, error) {
		now := time.Now()
		res := db.Model(&ItemEntity{}).
			Where("tenant_id = ? AND id = ? AND returning_at IS NULL", tenantId, id).
			Update("returning_at", now)
		return res.RowsAffected > 0, res.Error
	}
}

// RestoreItem un-soft-deletes one escrow row — the compensating inverse of a
// release. Restoring a row that was never deleted, or that was HARD deleted, is
// a no-op.
//
// It also RELEASES the return claim, in the same statement. A restore means the
// return demonstrably did not happen — the orchestrator's reverse walk put the
// item back into custody because a later step of that unwind failed — so the row
// is once again an asset nobody is returning. Leaving it latched would make the
// boot sweep, which is the retry of last resort for exactly this case, skip it
// forever: the player's item, gone, with no error anywhere. Clearing it cannot
// resurrect the duplicate this claim exists to prevent, because the duplicate
// requires an accept that succeeded, and an accept that succeeded is never
// followed by a restore of its own release (the reverse walk removes the granted
// item first — see the orchestrator's compensator).
func RestoreItem(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		return db.Unscoped().
			Model(&ItemEntity{}).
			Where("tenant_id = ? AND id = ?", tenantId, id).
			Updates(map[string]interface{}{"deleted_at": nil, "returning_at": nil}).Error
	}
}

// RemoveItem HARD deletes one escrow row — the compensating inverse of an
// accept whose paired release later failed.
//
// It must be a hard delete: a soft-deleted row is restorable, and restoring a
// row whose item was already re-granted to its owner would let a later unwind
// deliver the same item a second time.
func RemoveItem(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		return db.Unscoped().Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&ItemEntity{}).Error
	}
}

// UpsertMeso records a participant's ABSOLUTE escrowed meso total for a room.
//
// REPLACE, never accumulate. Clientbound mode 16 assigns rather than adds
// (design §1.6), so atlas-trades debits the delta between the requested total
// and this figure; a row that summed would be refunded more than was ever
// debited, minting meso on cancel.
func UpsertMeso(db *gorm.DB, t tenant.Model) func(roomId uuid.UUID, ownerId character.Id, amount uint32) error {
	return func(roomId uuid.UUID, ownerId character.Id, amount uint32) error {
		now := time.Now()
		e := MesoEntity{
			Id:           uuid.New(),
			TenantId:     t.Id(),
			TenantRegion: t.Region(),
			TenantMajor:  t.MajorVersion(),
			TenantMinor:  t.MinorVersion(),
			RoomId:       roomId,
			OwnerId:      ownerId,
			Amount:       amount,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		return db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "room_id"}, {Name: "owner_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"amount": amount, "updated_at": now}),
		}).Create(&e).Error
	}
}

// ArmMesoStake records an in-flight award_mesos debit against a participant's
// escrow row BEFORE the saga that performs it is submitted, so a terminal
// status that arrives after the room is gone still has somewhere durable to
// resolve against (see MesoEntity's doc comment).
//
// It creates the row (Amount 0) if this is the participant's first stake in
// the room. If a stake is already armed, it is OVERWRITTEN rather than
// rejected: the player retyping the stake box submits a fresh saga before the
// prior one necessarily finished, and the newer stake is authoritative — the
// prior stakeId's eventual terminal status must become a no-op, which is
// exactly what CommitMesoStake/AbandonMesoStake's compare-and-set gives it
// once PendingStakeId has moved on.
//
// delta is the signed movement the saga is about to submit — the stake minus
// what is escrowed RIGHT NOW — and it is recorded here rather than recomputed
// when the stake resolves, because Amount can legitimately move underneath a
// still-armed stake (see MesoEntity).
func ArmMesoStake(db *gorm.DB, t tenant.Model) func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID, amount uint32, delta int32) error {
	return func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID, amount uint32, delta int32) error {
		now := time.Now()
		e := MesoEntity{
			Id:             uuid.New(),
			TenantId:       t.Id(),
			TenantRegion:   t.Region(),
			TenantMajor:    t.MajorVersion(),
			TenantMinor:    t.MinorVersion(),
			RoomId:         roomId,
			OwnerId:        ownerId,
			Amount:         0,
			PendingStakeId: stakeId,
			PendingAmount:  amount,
			PendingDelta:   delta,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		return db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "room_id"}, {Name: "owner_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"pending_stake_id": stakeId,
				"pending_amount":   amount,
				"pending_delta":    delta,
				"updated_at":       now,
			}),
		}).Create(&e).Error
	}
}

// CommitMesoStake resolves an in-flight stake into the committed escrow total
// — the durable counterpart of the room applying an award_mesos COMPLETED
// status.
//
// The match on PendingStakeId happens INSIDE the UPDATE's WHERE clause, not as
// a separate read-then-write, so two concurrent deliveries of the same (or a
// stale) terminal status cannot both observe "still armed" and both commit:
// only the delivery whose UPDATE actually matches a row moves Amount, and it
// clears PendingStakeId in the same statement so a second delivery finds
// nothing to match. This is also what makes a stale stakeId — one a later
// ArmMesoStake already superseded — a silent no-op instead of double-applying
// a debit the player no longer intends.
func CommitMesoStake(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
		res := db.Model(&MesoEntity{}).
			Where("tenant_id = ? AND room_id = ? AND owner_id = ? AND pending_stake_id = ?", tenantId, roomId, ownerId, stakeId).
			Updates(map[string]interface{}{
				"amount":           gorm.Expr("pending_amount"),
				"pending_stake_id": uuid.Nil,
				"pending_amount":   0,
				"pending_delta":    0,
				"updated_at":       time.Now(),
			})
		return res.RowsAffected > 0, res.Error
	}
}

// AbandonMesoStake clears an in-flight stake WITHOUT committing it into
// Amount — the durable counterpart of the room applying an award_mesos FAILED
// or CANCELLED status. Same single-UPDATE compare-and-set as CommitMesoStake,
// for the same reason: a stale or redelivered terminal status must be inert.
func AbandonMesoStake(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
		res := db.Model(&MesoEntity{}).
			Where("tenant_id = ? AND room_id = ? AND owner_id = ? AND pending_stake_id = ?", tenantId, roomId, ownerId, stakeId).
			Updates(map[string]interface{}{
				"pending_stake_id": uuid.Nil,
				"pending_amount":   0,
				"pending_delta":    0,
				"updated_at":       time.Now(),
			})
		return res.RowsAffected > 0, res.Error
	}
}

// DeleteMeso removes a participant's escrowed meso row for a room. Like
// DeleteItem, a no-match is success.
//
// Meso rows are HARD deleted: there is no compensating restore for them because
// the refund is itself an award_mesos, which the saga compensator reverses on
// its own terms.
func DeleteMeso(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) error {
	return func(roomId uuid.UUID, ownerId character.Id) error {
		return db.Unscoped().
			Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
			Delete(&MesoEntity{}).Error
	}
}
