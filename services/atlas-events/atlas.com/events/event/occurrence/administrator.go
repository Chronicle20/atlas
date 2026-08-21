package occurrence

import (
	"atlas-events/event/transition"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// ErrConcurrencyKeyTaken is returned by createFromSeed when the occurrence
// insert collides with the partial unique index on (tenant, concurrency_key)
// — an active, non-terminal occurrence already holds this gameplay slot
// (design §5.3 guard 3). An empty concurrency key is excluded from the
// constraint and never produces this error.
var ErrConcurrencyKeyTaken = errors.New("occurrence: concurrency key already claimed by an active occurrence")

// ErrAlreadyCompleted is returned by applyProgress (terminal branch) and
// signals that a losing racer must be told "already completed" rather than
// "not found" — the two completion paths (ApplyProgress's terminal branch and
// complete()) converge on the same guarded UPDATE and share this outcome.
var ErrAlreadyCompleted = errors.New("occurrence: already completed")

// createFromSeed inserts the occurrence, its map scope rows and the
// OCCURRENCE_CREATED transition in ONE transaction. There is no path by which
// the administrator can write an occurrence without its transition (FR-O6).
//
// The insert uses ON CONFLICT DO NOTHING rather than a raw driver-error
// check: it is portable across the Postgres production driver and the SQLite
// test driver, and RowsAffected == 0 is an unambiguous signal that the
// concurrency-key index rejected the row.
func createFromSeed(db *gorm.DB) func(entity Entity, maps []MapEntity, trans transition.Entity) (Entity, error) {
	return func(entity Entity, maps []MapEntity, trans transition.Entity) (Entity, error) {
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&entity)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrConcurrencyKeyTaken
			}
			if len(maps) > 0 {
				if err := tx.Create(&maps).Error; err != nil {
					return err
				}
			}
			return tx.Create(&trans).Error
		})
		if err != nil {
			return Entity{}, err
		}
		return entity, nil
	}
}

// applyProgress updates the occurrence's state/stage/context/next-transition
// (and, when terminal, completion) and writes the paired transition row in ONE
// transaction (FR-O6/FR-T2).
//
// terminal selects the guard: a non-terminal write only needs the row to
// exist (WHERE id = ?), and a nonexistent id surfaces gorm.ErrRecordNotFound.
// A terminal write (this call is completing the occurrence) uses the SAME
// guarded UPDATE complete() uses — WHERE id = ? AND state = 'ACTIVE' — so it
// converges on the one guarded completion transition (design §686). A losing
// racer (RowsAffected == 0) is reported as ErrAlreadyCompleted, distinct from
// "no such occurrence", and writes no transition row.
func applyProgress(db *gorm.DB) func(entity Entity, trans transition.Entity, terminal bool) (Entity, error) {
	return func(entity Entity, trans transition.Entity, terminal bool) (Entity, error) {
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			updates := map[string]interface{}{
				"state":              entity.State,
				"stage":              entity.Stage,
				"context":            entity.Context,
				"next_transition_at": entity.NextTransitionAt,
				"completed_at":       entity.CompletedAt,
				"completion_reason":  entity.CompletionReason,
			}
			q := tx.Model(&Entity{}).Where("id = ?", entity.ID)
			if terminal {
				q = q.Where("state = ?", StateActive)
			}
			res := q.Updates(updates)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				if terminal {
					return ErrAlreadyCompleted
				}
				return gorm.ErrRecordNotFound
			}
			return tx.Create(&trans).Error
		})
		if err != nil {
			return Entity{}, err
		}
		return entity, nil
	}
}

// observeMonsterSpawned is INSERT-IF-ABSENT, deliberately not an upsert: a
// KILLED that arrived before its CREATED already wrote a dead row, and the
// late CREATED must not resurrect it. The two events share a topic but have
// no ordering guarantee across partitions, so this is a real case (design
// §9.5, FR-B18).
func observeMonsterSpawned(db *gorm.DB) func(entity MonsterEntity) error {
	return func(entity MonsterEntity) error {
		return db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "occurrence_id"}, {Name: "unique_id"}},
			DoNothing: true,
		}).Create(&entity).Error
	}
}

// observeMonsterGone is an UPSERT to alive=false: idempotent by construction,
// and correct whether or not CREATED was seen first.
func observeMonsterGone(db *gorm.DB) func(entity MonsterEntity) error {
	return func(entity MonsterEntity) error {
		return db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "occurrence_id"}, {Name: "unique_id"}},
			DoUpdates: clause.Assignments(map[string]any{"alive": false, "observed_at": entity.ObservedAt}),
		}).Create(&entity).Error
	}
}

// complete is a GUARDED update, not a lock: the completion decision itself is
// the WHERE state = 'ACTIVE' predicate on the UPDATE below, not the SELECT.
// RowsAffected == 0 means another path completed this occurrence first; the
// caller must then skip its cleanup and return success. This is FR-B20
// expressed as a database predicate. The transition row is written ONLY on
// the winning path — the loser makes no state change, so it must write no
// transition (FR-O6/FR-T2).
//
// The row is read once, with a SELECT ... FOR UPDATE, before the guarded
// UPDATE, so the transition's FromStage records the stage that was actually
// true at write time rather than one read outside — and possibly stale by —
// the transaction (buildTrans receives it). SELECT ... FOR UPDATE is a no-op
// under the SQLite test driver (it does not support row locking) and under
// Postgres it holds the row for the remainder of this transaction, so the
// stage buildTrans sees cannot change again before the guarded UPDATE
// commits. A nonexistent id surfaces gorm.ErrRecordNotFound from the SELECT.
func complete(db *gorm.DB) func(id uuid.UUID, reason string, at time.Time, buildTrans func(fromStage string) (transition.Entity, error)) (bool, error) {
	return func(id uuid.UUID, reason string, at time.Time, buildTrans func(fromStage string) (transition.Entity, error)) (bool, error) {
		var won bool
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			var current Entity
			if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
				Where("id = ?", id).First(&current).Error; err != nil {
				return err
			}

			res := tx.Model(&Entity{}).
				Where("id = ? AND state = ?", id, StateActive).
				Updates(map[string]any{
					"state":             StateCompleted,
					"completion_reason": reason,
					"completed_at":      at,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				won = false
				return nil
			}
			won = true
			trans, err := buildTrans(current.Stage)
			if err != nil {
				return err
			}
			return tx.Create(&trans).Error
		})
		return won, err
	}
}
