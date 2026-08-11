package escrow

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
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
func ClaimItemForReturn(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID, txId uuid.UUID) (bool, error) {
	return func(id uuid.UUID, txId uuid.UUID) (bool, error) {
		now := time.Now()
		res := db.Model(&ItemEntity{}).
			Where("tenant_id = ? AND id = ? AND returning_at IS NULL", tenantId, id).
			Updates(map[string]interface{}{"returning_at": now, "returning_tx_id": txId})
		return res.RowsAffected > 0, res.Error
	}
}

// ReleaseItemReturnClaims un-latches every row one trade_unwind claimed, so a
// FAILED unwind hands its rows back to whatever tries next instead of stranding
// them.
//
// Without it a failed unwind left its rows latched forever: the latch clears
// only on a completed release, and the boot sweep skips a latched row by design
// — so the item sat intact in custody, owned by nobody, invisible to every path
// that could have returned it.
//
// Scoped to the transaction, never a blanket clear: rows another unwind is
// legitimately returning must keep their claims.
func ReleaseItemReturnClaims(db *gorm.DB, tenantId uuid.UUID) func(txId uuid.UUID) (int64, error) {
	return func(txId uuid.UUID) (int64, error) {
		res := db.Model(&ItemEntity{}).
			Where("tenant_id = ? AND returning_tx_id = ?", tenantId, txId).
			Updates(map[string]interface{}{"returning_at": nil, "returning_tx_id": nil})
		return res.RowsAffected, res.Error
	}
}

// RecordMesoRefund durably names what one trade_unwind is taking from a
// participant, in the transaction that takes it. See MesoRefundEntity.
func RecordMesoRefund(db *gorm.DB, t tenant.Model) func(txId uuid.UUID, roomId uuid.UUID, ownerId character.Id, amount int64) error {
	return func(txId uuid.UUID, roomId uuid.UUID, ownerId character.Id, amount int64) error {
		return db.Create(&MesoRefundEntity{
			Id:            uuid.New(),
			TransactionId: txId,
			TenantId:      t.Id(),
			TenantRegion:  t.Region(),
			TenantMajor:   t.MajorVersion(),
			TenantMinor:   t.MinorVersion(),
			RoomId:        roomId,
			OwnerId:       ownerId,
			Amount:        amount,
			CreatedAt:     time.Now(),
		}).Error
	}
}

// RestoreMesoRefunds puts back everything one FAILED trade_unwind took, and
// reports how many participants it restored.
//
// The escrow row is re-created if the retire already removed it — a claim that
// emptied a row typically retires it in the same pass — because the amount has
// to land somewhere the boot sweep will find. The add is RELATIVE, so a stake
// that committed in the meantime is not clobbered.
//
// The records are consumed in the same transaction, making a redelivered
// failure inert: the second pass finds nothing to restore.
func RestoreMesoRefunds(db *gorm.DB, tenantId uuid.UUID) func(txId uuid.UUID) (int, error) {
	return func(txId uuid.UUID) (int, error) {
		restored := 0
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			var rows []MesoRefundEntity
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("tenant_id = ? AND transaction_id = ?", tenantId, txId).
				Find(&rows).Error; err != nil {
				return err
			}
			if len(rows) == 0 {
				return nil
			}
			for _, r := range rows {
				e := MesoEntity{
					Id: uuid.New(), TenantId: r.TenantId, TenantRegion: r.TenantRegion,
					TenantMajor: r.TenantMajor, TenantMinor: r.TenantMinor,
					RoomId: r.RoomId, OwnerId: r.OwnerId, Amount: r.Amount,
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "room_id"}, {Name: "owner_id"}},
					DoUpdates: clause.Assignments(map[string]interface{}{"amount": gorm.Expr(mesoTable+".amount + ?", r.Amount), "updated_at": time.Now()}),
				}).Create(&e).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("tenant_id = ? AND transaction_id = ?", tenantId, txId).
				Delete(&MesoRefundEntity{}).Error; err != nil {
				return err
			}
			restored = len(rows)
			return nil
		})
		return restored, err
	}
}

// DiscardMesoRefunds drops the records of a SUCCEEDED trade_unwind: the meso
// reached the player, so there is nothing left to put back.
func DiscardMesoRefunds(db *gorm.DB, tenantId uuid.UUID) func(txId uuid.UUID) (int64, error) {
	return func(txId uuid.UUID) (int64, error) {
		res := db.Where("tenant_id = ? AND transaction_id = ?", tenantId, txId).Delete(&MesoRefundEntity{})
		return res.RowsAffected, res.Error
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

// UpsertMeso ASSIGNS a participant's confirmed escrowed meso total for a room.
//
// It is the blunt setter, used by teardown paths that know the whole custody is
// over (see the trade package's clearRefundedMesos, which zeroes a refunded row
// while deliberately leaving its stakes armed). Ordinary staging never calls it:
// a confirmed stake advances the total by ADDING its own delta inside
// CommitMesoStake, which is what lets several stakes resolve in any order
// without clobbering each other. Assigning from a figure read a moment earlier
// would lose whichever sibling committed in between.
func UpsertMeso(db *gorm.DB, t tenant.Model) func(roomId uuid.UUID, ownerId character.Id, amount int64) error {
	return func(roomId uuid.UUID, ownerId character.Id, amount int64) error {
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
// resolve against (see MesoStakeEntity's doc comment).
//
// It creates the owning meso row (Amount 0) if this is the participant's first
// stake in the room, then inserts the stake as a row of its own. A stake
// already in flight is left ALONE rather than overwritten: its debit moved real
// meso, so it still has to resolve on its own terms. Both stakes are
// outstanding together and each commits its own delta — which is what makes the
// arithmetic conserve when a player retypes the box mid-saga.
//
// delta is the signed movement the saga is about to submit — the target minus
// committed PLUS whatever is already in flight — and it is recorded here rather
// than recomputed when the stake resolves, because Amount is not stable across
// the stake's lifetime (see MesoStakeEntity).
//
// The two writes share one transaction: a stake row whose owning meso row was
// never created would be resolvable but not readable by the sweep that has to
// find stranded custody.
func ArmMesoStake(db *gorm.DB, t tenant.Model) func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID, amount uint32, delta int32) error {
	return func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID, amount uint32, delta int32) error {
		now := time.Now()
		return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			owner := MesoEntity{
				Id:           uuid.New(),
				TenantId:     t.Id(),
				TenantRegion: t.Region(),
				TenantMajor:  t.MajorVersion(),
				TenantMinor:  t.MinorVersion(),
				RoomId:       roomId,
				OwnerId:      ownerId,
				Amount:       0,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			// DoNothing, not an assignment: the owning row's Amount is the
			// committed total and arming a stake must never disturb it.
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "room_id"}, {Name: "owner_id"}},
				DoNothing: true,
			}).Create(&owner).Error; err != nil {
				return err
			}
			return tx.Create(&MesoStakeEntity{
				Id:           stakeId,
				TenantId:     t.Id(),
				TenantRegion: t.Region(),
				TenantMajor:  t.MajorVersion(),
				TenantMinor:  t.MinorVersion(),
				RoomId:       roomId,
				OwnerId:      ownerId,
				Amount:       amount,
				Delta:        delta,
				CreatedAt:    now,
			}).Error
		})
	}
}

// CommitMesoStake folds a resolved stake's delta into the committed escrow
// total — the durable counterpart of the room applying an award_mesos COMPLETED
// status. It reports whether this call is the one that claimed the stake.
//
// The DELETE of the stake row IS the compare-and-set, and it is what makes the
// operation idempotent: two concurrent or redelivered terminal statuses race to
// delete one row, exactly one gets RowsAffected 1, and only that one adds the
// delta. The loser sees no row and reports false. Nothing here reads the stake
// first and acts on it afterwards — under READ COMMITTED (the isolation this
// fleet runs at, verified against the live cluster) a read-then-write pair
// would let both deliveries observe "still armed" and both commit the debit.
//
// Amount is advanced by `amount + delta` as a SQL expression rather than by
// assigning a figure computed in Go, for the same reason: the committed total
// can move underneath this statement — a concurrent sibling stake committing
// its own delta, or a teardown zeroing the row — and a read-modify-write would
// clobber whichever landed in between. Addition of one's own delta is the only
// form that composes with the others.
//
// A stake row that no longer exists is not an error. It means some other
// delivery already resolved it, which is the ordinary shape of an
// at-least-once terminal status.
func CommitMesoStake(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
		claimed := false
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			var stake MesoStakeEntity
			if err := tx.Clauses(clause.Returning{}).
				Where("id = ? AND tenant_id = ? AND room_id = ? AND owner_id = ?", stakeId, tenantId, roomId, ownerId).
				Delete(&stake).Error; err != nil {
				return err
			}
			if stake.Id == uuid.Nil {
				return nil
			}
			claimed = true
			// A stake whose delta is zero still has to be claimed — the delete
			// above did that — but it moves nothing, so skip the UPDATE.
			if stake.Delta == 0 {
				return nil
			}
			return tx.Model(&MesoEntity{}).
				Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
				Updates(map[string]interface{}{
					"amount":     gorm.Expr("amount + ?", stake.Delta),
					"updated_at": time.Now(),
				}).Error
		})
		return claimed, err
	}
}

// AbandonMesoStake discards an in-flight stake WITHOUT moving Amount — the
// durable counterpart of the room applying an award_mesos FAILED or CANCELLED
// status.
//
// Amount is untouched because a failed stake's delta never moved: the saga's
// own compensator returned it. Committing it here would credit escrow nobody
// paid for. Same single-statement claim as CommitMesoStake, for the same
// reason: a stale or redelivered terminal status must be inert.
func AbandonMesoStake(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id, stakeId uuid.UUID) (bool, error) {
		res := db.Where("id = ? AND tenant_id = ? AND room_id = ? AND owner_id = ?", stakeId, tenantId, roomId, ownerId).
			Delete(&MesoStakeEntity{})
		return res.RowsAffected > 0, res.Error
	}
}

// ClaimMesoForReturn takes a participant's whole escrowed total for refund and
// reports what THIS caller won, zero if another path got there first. It is the
// meso twin of ClaimItemForReturn.
//
// Two independent paths can each decide to refund one row: a room teardown
// reading MesosByRoom, and the boot/ticker sweep reading AllMesos. Both used to
// build their unwind from a total they had READ and only zero the row
// afterwards, so under READ COMMITTED both could read the same total and both
// submit a refund for it. Nothing downstream dedupes them — a meso unwind leg is
// a bare award_mesos credit, with no custody row to be already gone — so the
// player was paid twice.
//
// Read-and-take is therefore made indivisible by taking an exclusive ROW LOCK
// on the SELECT (`FOR UPDATE`) and zeroing inside the same transaction. A
// competitor's claim blocks on the lock until this transaction commits, and
// then reads the zero — so the amount is handed to exactly one caller. A
// RETURNING clause cannot do this on its own: an UPDATE returns the row as it
// stands AFTER the assignment, which is the zero, not the amount taken.
//
// The caller must submit the refund in the same transaction that claims, which
// every caller does by running inside emit's — otherwise a crash between the two
// loses the claimed meso with no record that it was ever owed.
//
// The row SURVIVES, zeroed, rather than being deleted: a stake still in flight
// resolves against it, and deleting it would strand a debit the player has
// already been charged. Retiring it is DeleteResolvedMeso's job, conditional on
// there being no stake left.
//
// A non-positive row yields nothing. Zero holds no custody, and negative means
// more reduction has been confirmed than increase so far (see MesoEntity).
func ClaimMesoForReturn(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) (int64, bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id) (int64, bool, error) {
		var claimed int64
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			var e MesoEntity
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
				First(&e).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			if e.Amount <= 0 {
				return nil
			}
			res := tx.Model(&MesoEntity{}).
				Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
				Updates(map[string]interface{}{"amount": 0, "updated_at": time.Now()})
			if res.Error != nil {
				return res.Error
			}
			claimed = e.Amount
			return nil
		})
		return claimed, claimed > 0, err
	}
}

// DischargeMeso SUBTRACTS an amount from a participant's confirmed escrowed
// total — used when custody of that amount has demonstrably ended, i.e. it has
// just been handed back to the player by an unwind.
//
// Relative, and applied by the database, for the reason CommitMesoStake gives:
// the total can move underneath this statement, and an assignment computed from
// a figure read beforehand would clobber whichever concurrent write landed in
// between. There is deliberately no floor — Amount is signed and a transient
// negative is legitimate (see MesoEntity) — so clamping here would silently
// swallow the very arithmetic error this is meant to keep honest.
func DischargeMeso(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id, amount int32) error {
	return func(roomId uuid.UUID, ownerId character.Id, amount int32) error {
		return db.Model(&MesoEntity{}).
			Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
			Updates(map[string]interface{}{
				"amount":     gorm.Expr("amount - ?", amount),
				"updated_at": time.Now(),
			}).Error
	}
}

// DeleteMeso removes a participant's escrowed meso row for a room. Like
// DeleteItem, a no-match is success.
//
// Meso rows are HARD deleted: there is no compensating restore for them because
// the refund is itself an award_mesos, which the saga compensator reverses on
// its own terms.
//
// Any in-flight stakes go with the row, in the same transaction. A stake left
// behind by a deleted owner row is unresolvable — its commit would add a delta
// to a row that no longer exists — and invisible to the boot sweep, which walks
// owner rows. Callers that mean to keep stakes alive across a teardown zero the
// row with UpsertMeso instead (see the trade package's clearRefundedMesos);
// this call means the custody itself is over.
func DeleteMeso(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) error {
	return func(roomId uuid.UUID, ownerId character.Id) error {
		return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			if err := tx.Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
				Delete(&MesoStakeEntity{}).Error; err != nil {
				return err
			}
			return tx.Unscoped().
				Where("tenant_id = ? AND room_id = ? AND owner_id = ?", tenantId, roomId, ownerId).
				Delete(&MesoEntity{}).Error
		})
	}
}

// DeleteResolvedMeso removes a participant's escrow meso row ONLY IF it is fully
// resolved — nothing escrowed and no stake in flight — and reports whether a row
// actually went.
//
// A meso row exists to record custody. Once Amount is zero and PendingStakeId is
// the none sentinel it records none: no unwind will read it, no terminal status
// will resolve against it, and every later boot sweep pays to read it again
// (AllMesos is unfiltered, so that cost grows with lifetime trade volume). Worse,
// a row left behind carrying a STALE non-zero total is read by the boot sweep as
// a stranded asset and refunded a second time, so retiring the row is a
// correctness rule and not only a housekeeping one.
//
// Both halves of "resolved" are tested INSIDE the DELETE's WHERE clause rather
// than by a read the caller made first, for the same reason CommitMesoStake's
// compare-and-set is: a stage that arms a fresh stake against this row races
// every caller here, and a read-then-delete would drop the row the arming stage
// is relying on — stranding a debit the player has already been charged. Under
// the statement, an arm that lands first simply makes the delete match nothing,
// and an arm that lands second re-creates the row it needs (ArmMesoStake is an
// upsert).
//
// The delete is unconditionally HARD, like DeleteMeso: MesoEntity carries no
// soft-delete column and there is nothing to compensate.
func DeleteResolvedMeso(db *gorm.DB, tenantId uuid.UUID) func(roomId uuid.UUID, ownerId character.Id) (bool, error) {
	return func(roomId uuid.UUID, ownerId character.Id) (bool, error) {
		// "No stake in flight" is now a NOT EXISTS against the stake table
		// rather than a sentinel comparison, but it remains INSIDE the DELETE's
		// WHERE clause for the reason above: a stage arming a stake races every
		// caller here, and a read-then-delete would drop the row that stage
		// depends on.
		res := db.Where(`tenant_id = ? AND room_id = ? AND owner_id = ? AND amount = 0
			AND NOT EXISTS (SELECT 1 FROM `+mesoStakeTable+` s WHERE s.tenant_id = `+mesoTable+`.tenant_id AND s.room_id = `+mesoTable+`.room_id AND s.owner_id = `+mesoTable+`.owner_id)`,
			tenantId, roomId, ownerId).
			Delete(&MesoEntity{})
		return res.RowsAffected > 0, res.Error
	}
}
