package parcel

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Create persists a new parcel and returns the stored Model. TenantId is
// injected by the atlas-database tenant:create callback from the context on
// db when the caller has not set it — this administrator must NOT set
// tenant_id itself.
func Create(db *gorm.DB) func(m Model) (Model, error) {
	return func(m Model) (Model, error) {
		e := entityFromModel(m)
		if e.Id == uuid.Nil {
			e.Id = uuid.New()
		}
		err := db.Create(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

// UpdateStatus transitions a parcel to status and stamps resolvedAt (the
// receive/discard/expire timestamp).
func UpdateStatus(db *gorm.DB) func(id uuid.UUID, status string, resolvedAt time.Time) error {
	return func(id uuid.UUID, status string, resolvedAt time.Time) error {
		return db.Model(&Entity{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":      status,
				"resolved_at": resolvedAt,
			}).Error
	}
}

// UpdateStatusIfPending is UpdateStatus's compare-and-swap sibling: it only
// applies the transition when the row is still status='pending' at write
// time, and reports how many rows it actually changed (0 or 1). Processor's
// Receive/Discard use this — not UpdateStatus — so that a concurrent
// duplicate delivery cannot both succeed: the losing writer's UPDATE affects
// zero rows even though its own in-transaction read (moments earlier) still
// saw status='pending'. UpdateStatus itself is left unpredicated for
// task-23's expiry/return sweep, which transitions FROM a status it already
// holds under its own claim query, not from a caller-asserted "was pending."
func UpdateStatusIfPending(db *gorm.DB) func(id uuid.UUID, status string, resolvedAt time.Time) (int64, error) {
	return func(id uuid.UUID, status string, resolvedAt time.Time) (int64, error) {
		res := db.Model(&Entity{}).
			Where("id = ? AND status = ?", id, StatusPending).
			Updates(map[string]interface{}{
				"status":      status,
				"resolved_at": resolvedAt,
			})
		return res.RowsAffected, res.Error
	}
}

// ClaimExpired is the expiry sweep's claim-by-update (design §8.1, NFR-7):
// one UPDATE that both selects and transitions up to batch expired-pending
// rows in a single statement, guarded by status='pending' both in the outer
// WHERE and the candidate subquery. That outer guard is the actual
// concurrency safety property — under concurrent replicas, whichever
// UPDATE's row-level write commits first is the only one whose WHERE
// clause still matches that row by the time it runs; the loser's UPDATE
// simply claims zero of that row, no leader election required. LIMIT
// cannot appear directly on an UPDATE (unsupported by both postgres and
// sqlite's default build), so the batch cap lives in a SELECT id ... LIMIT
// subquery instead — standard SQL, portable to both backends this service
// runs under (production postgres, databasetest's in-memory sqlite).
func ClaimExpired(db *gorm.DB) func(now time.Time, batch int) ([]Model, error) {
	return func(now time.Time, batch int) ([]Model, error) {
		candidates := db.Model(&Entity{}).
			Select("id").
			Where("status = ? AND expires_at <= ?", StatusPending, now).
			Order("expires_at ASC").
			Limit(batch)

		var entities []Entity
		err := db.Clauses(clause.Returning{}).
			Model(&entities).
			Where("status = ? AND expires_at <= ?", StatusPending, now).
			Where("id IN (?)", candidates).
			Updates(map[string]interface{}{
				"status":      StatusExpired,
				"resolved_at": now,
			}).Error
		if err != nil {
			return nil, err
		}
		return model.SliceMap(Make)(model.FixedProvider(entities))()()
	}
}

// StampNotified records LastNotified for a batch of parcels — used by the
// mailbox notification sweep (task-24).
func StampNotified(db *gorm.DB) func(ids []uuid.UUID, at time.Time) error {
	return func(ids []uuid.UUID, at time.Time) error {
		if len(ids) == 0 {
			return nil
		}
		return db.Model(&Entity{}).
			Where("id IN ?", ids).
			Update("last_notified", at).Error
	}
}

// ClaimNotifiable is the notification sweep's claim-by-update (design §8.1,
// last paragraph: "the notification sweep uses the same claim-by-update
// shape on LastNotified"): one UPDATE both selects and stamps up to batch
// newly-receivable, not-yet-notified rows. Stamping LastNotified as part of
// the SAME statement that claims the row is the concurrency guard — exactly
// ClaimExpired's shape, with LastNotified IS NULL standing in for
// status='pending' as the compare-and-swap predicate. Under concurrent
// replicas, whichever UPDATE's write commits first is the only one whose
// WHERE clause (last_notified IS NULL) still matches that row; the loser
// claims zero of it, so at most one Kafka PARCEL_ARRIVED event is ever
// emitted per parcel.
func ClaimNotifiable(db *gorm.DB) func(now time.Time, batch int) ([]Model, error) {
	return func(now time.Time, batch int) ([]Model, error) {
		candidates := db.Model(&Entity{}).
			Select("id").
			Where("last_notified IS NULL AND status = ? AND receivable_at <= ?", StatusPending, now).
			Order("receivable_at ASC").
			Limit(batch)

		var entities []Entity
		err := db.Clauses(clause.Returning{}).
			Model(&entities).
			Where("last_notified IS NULL AND status = ? AND receivable_at <= ?", StatusPending, now).
			Where("id IN (?)", candidates).
			Update("last_notified", now).Error
		if err != nil {
			return nil, err
		}
		return model.SliceMap(Make)(model.FixedProvider(entities))()()
	}
}
