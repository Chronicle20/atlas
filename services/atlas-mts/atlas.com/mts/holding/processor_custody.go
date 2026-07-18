package holding

import (
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// ReleaseResult reports the outcome of a Release. Taken is the holding captured
// BEFORE the soft-delete (its owner/world/item), and EmitTakenHome is true iff the
// row was present (a first delivery) — the consumer uses it to gate the
// ITEM_TAKEN_HOME event so a replay (row already gone) does not re-emit.
type ReleaseResult struct {
	Taken         Model
	EmitTakenHome bool
}

// Release soft-deletes the holding row by id. Soft-delete is idempotent: a replayed
// delivery affects 0 rows (already gone) and is success. The holding's
// owner/world/item is captured BEFORE the soft-delete so the take-home
// ITEM_TAKEN_HOME event can address the owner's Transfer Inventory re-push; a miss
// means the row is already soft-deleted (a replay), so EmitTakenHome is false and
// the consumer skips the re-emit. The whole delete runs in one local DB transaction.
func (p *ProcessorImpl) Release(holdingId string) (ReleaseResult, error) {
	tdb := p.db.WithContext(p.ctx)

	// Capture the holding's owner/world/item BEFORE the soft-delete so the
	// take-home ITEM_TAKEN_HOME event can address the owner's Transfer
	// Inventory re-push. A miss means the row is already soft-deleted (a
	// replay), so the event was emitted on the first delivery — skip the
	// re-emit to keep release idempotent.
	var res ReleaseResult
	err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		if hm, gerr := GetById(holdingId)(tx)(); gerr == nil {
			res.Taken = hm
			res.EmitTakenHome = true
		}
		// SoftDelete is idempotent: 0 rows affected on a replay (already
		// released) is success, not an error.
		_, derr := SoftDelete(tx, holdingId)
		return derr
	})
	if err != nil {
		return ReleaseResult{}, err
	}
	return res, nil
}

// RestoreHolding un-soft-deletes the holding row by id — the inverse of Release,
// dispatched by the saga compensator when a WithdrawFromMts saga fails after the
// holding was already released. Restore is idempotent: clearing deleted_at on an
// already-live row affects 0 rows and is success. The whole restore runs in one
// local DB transaction.
func (p *ProcessorImpl) RestoreHolding(holdingId string) error {
	tdb := p.db.WithContext(p.ctx)

	return database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		// Restore is idempotent: 0 rows affected on a replay (already
		// live) is success, not an error.
		_, rerr := Restore(tx, holdingId)
		return rerr
	})
}
