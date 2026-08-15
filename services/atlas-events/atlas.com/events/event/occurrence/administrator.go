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
// transaction (FR-O6/FR-T2). A nonexistent id surfaces gorm.ErrRecordNotFound
// rather than silently no-oping.
func applyProgress(db *gorm.DB) func(entity Entity, trans transition.Entity) (Entity, error) {
	return func(entity Entity, trans transition.Entity) (Entity, error) {
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			updates := map[string]interface{}{
				"state":              entity.State,
				"stage":              entity.Stage,
				"context":            entity.Context,
				"next_transition_at": entity.NextTransitionAt,
				"completed_at":       entity.CompletedAt,
				"completion_reason":  entity.CompletionReason,
			}
			res := tx.Model(&Entity{}).Where("id = ?", entity.ID).Updates(updates)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
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

// complete is a GUARDED update, not a lock. RowsAffected == 0 means another
// path completed this occurrence first; the caller must then skip its cleanup
// and return success. This is FR-B20 expressed as a database predicate. The
// transition row is written ONLY on the winning path — the loser makes no
// state change, so it must write no transition (FR-O6/FR-T2).
func complete(db *gorm.DB) func(id uuid.UUID, reason string, at time.Time, trans transition.Entity) (bool, error) {
	return func(id uuid.UUID, reason string, at time.Time, trans transition.Entity) (bool, error) {
		var won bool
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
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
			return tx.Create(&trans).Error
		})
		return won, err
	}
}
