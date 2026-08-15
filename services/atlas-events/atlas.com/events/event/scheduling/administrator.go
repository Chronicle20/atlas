package scheduling

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Administrator is the entry point onto scheduled event work persistence.
type Administrator struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

// NewAdministrator constructs an Administrator scoped to the tenant carried
// by ctx.
func NewAdministrator(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Administrator {
	return &Administrator{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		db:  db,
	}
}

// Schedule inserts m as a new scheduled work row and returns it with
// created=true. If m carries a non-empty DedupeKey and a PENDING or
// PROCESSING row already exists with that key, the insert is skipped and the
// existing row is returned instead, with created=false and a nil error — a
// redelivered Kafka message that schedules the same logical work twice must
// be a no-op, not a second row and not an error (FR-B4/FR-S8). An empty
// DedupeKey opts a row out of dedup entirely.
//
// The dedupe check reads-then-inserts inside a transaction, then the insert
// itself uses `ON CONFLICT DO NOTHING` (no target) rather than letting a
// conflicting insert fail: on Postgres — the production driver
// (libs/atlas-database/connection.go) — a failed statement poisons the rest
// of the transaction ("current transaction is aborted"), so a naive
// insert-then-catch-and-requery would fail its re-read on the very same
// handle once Task 19 adds the partial unique index ux_sew_dedupe. GORM
// signals a losing conflict via RowsAffected == 0 instead of a driver error,
// so the transaction stays live and the re-read below succeeds. This is the
// same idiom event/occurrence's createFromSeed already uses for its own
// concurrency-key race (event/occurrence/administrator.go). Until Task 19
// adds the index, no constraint exists to conflict with, so this clause is
// inert and the app-level pre-check above is the only guard — Task 19 needs
// to add nothing beyond the index itself.
func (a *Administrator) Schedule(m Model) (Model, bool, error) {
	db := a.db.WithContext(a.ctx)

	var result Model
	var created bool

	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		if m.DedupeKey() != "" {
			if existing, ok, err := findActiveDedupe(tx, m.DedupeKey()); err != nil {
				return err
			} else if ok {
				result = existing
				created = false
				return nil
			}
		}

		// Persistence assigns the row's identity fresh on every actual insert,
		// rather than trusting m.Id() — a retried Schedule call for the same
		// logical work (e.g. the same Model value re-submitted after its
		// prior row was cancelled, as in a redelivery-after-cancel) must
		// produce a genuinely new row, not collide on a stale PK.
		toInsert, err := m.Builder().SetId(uuid.New()).Build()
		if err != nil {
			return err
		}
		entity, err := ToEntity(toInsert, a.t.Id())
		if err != nil {
			return err
		}

		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&entity)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Lost the race: some other Schedule call won the dedupe key
			// between our pre-check and this insert. Re-read the winner on
			// this SAME (still-live) transaction handle.
			existing, ok, err := findActiveDedupe(tx, m.DedupeKey())
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("scheduling: insert conflicted but no active dedupe row for key [%s]", m.DedupeKey())
			}
			result = existing
			created = false
			return nil
		}

		made, err := Make(entity)
		if err != nil {
			return err
		}
		result = made
		created = true
		return nil
	})
	if err != nil {
		a.l.WithError(err).Errorf("Failed to schedule event work of type [%s].", m.Type())
		return Model{}, false, err
	}
	return result, created, nil
}

// findActiveDedupe looks up the PENDING or PROCESSING row matching dedupeKey.
// ok is false (with a nil error) when no such row exists.
func findActiveDedupe(db *gorm.DB, dedupeKey string) (Model, bool, error) {
	entity, err := getActiveByDedupeKeyProvider(dedupeKey)(db)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Model{}, false, nil
		}
		return Model{}, false, err
	}
	m, err := Make(entity)
	if err != nil {
		return Model{}, false, err
	}
	return m, true, nil
}

// SetState transitions the work row identified by id to newState, recording
// lastError when non-empty. It checks RowsAffected and surfaces
// gorm.ErrRecordNotFound for a missing id rather than silently no-oping.
func (a *Administrator) SetState(id uuid.UUID, newState string, lastError string) (Model, error) {
	db := a.db.WithContext(a.ctx)

	updates := map[string]interface{}{"state": newState}
	if lastError != "" {
		updates["last_error"] = lastError
	}

	result := db.Model(&Entity{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		a.l.WithError(result.Error).Errorf("Failed to set state on scheduled event work [%s].", id)
		return Model{}, result.Error
	}
	if result.RowsAffected == 0 {
		return Model{}, gorm.ErrRecordNotFound
	}

	entity, err := getByIdProvider(id)(db)()
	if err != nil {
		return Model{}, err
	}
	return Make(entity)
}

// CancelPendingForDefinition cancels every PENDING row belonging to
// definitionId — e.g. an Anniversary definition whose start time is edited
// before it fires (FR-S10). A row already PROCESSING belongs to a claimer
// and is left alone. It returns the number of rows cancelled; zero is a
// legitimate result (a bulk predicate cancel, not a lookup by id), so unlike
// SetState it does not check RowsAffected against gorm.ErrRecordNotFound.
func CancelPendingForDefinition(db *gorm.DB) func(definitionId uuid.UUID) (int64, error) {
	return func(definitionId uuid.UUID) (int64, error) {
		result := db.Model(&Entity{}).
			Where("event_definition_id = ? AND state = ?", definitionId, StatePending).
			Update("state", StateCancelled)
		return result.RowsAffected, result.Error
	}
}
