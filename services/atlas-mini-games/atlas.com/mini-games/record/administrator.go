package record

import (
	"errors"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getOrCreate returns the existing game_records row for (characterId,
// gameType), creating a zero-valued one if absent. Tenancy is context-driven
// (DOM-11): the caller passes a db already scoped with WithContext(ctx) so the
// tenant callbacks (database.RegisterTenantCallbacks) add the tenant_id filter
// on the SELECT and inject tenant_id on the INSERT. Callers running inside a
// transaction must pass the tx *gorm.DB so the create is part of the same
// transaction.
func getOrCreate(db *gorm.DB, characterId uint32, gameType GameType) (Entity, error) {
	var e Entity
	err := db.Where("character_id = ? AND game_type = ?", characterId, string(gameType)).First(&e).Error
	if err == nil {
		return e, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Entity{}, err
	}

	// TenantId is intentionally left zero: the tenant:create callback injects
	// it from context on INSERT (injectTenantIdIfZero).
	e = Entity{
		Id:          uuid.New(),
		CharacterId: characterId,
		GameType:    string(gameType),
	}
	if err := db.Create(&e).Error; err != nil {
		return Entity{}, err
	}
	return e, nil
}

// ApplyResult upserts both the owner's and visitor's game_records row for a
// finished game, inside a single transaction. winnerSlot 0 means the owner
// won (visitor lost), winnerSlot 1 means the visitor won (owner lost); tie
// overrides winnerSlot and increments both sides' Ties.
//
// Atomicity of the two-row update is via database.ExecuteTransaction (task-119
// fixed its formerly no-op behavior): it opens a transaction when db is not
// already in one, and JOINS the caller's transaction when it is — so the
// endGame outbox path can pass its own tx and have the record writes enqueue in
// the same transaction as the record-carrying events. Tenancy is context-driven
// (DOM-11): the caller passes a WithContext(ctx)-scoped db and the tenant
// callbacks scope both getOrCreate reads and writes inside the transaction.
func ApplyResult(db *gorm.DB, gameType GameType, ownerId uint32, visitorId uint32, winnerSlot byte, tie bool) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		or, err := getOrCreate(tx, ownerId, gameType)
		if err != nil {
			return err
		}
		vr, err := getOrCreate(tx, visitorId, gameType)
		if err != nil {
			return err
		}

		if tie {
			or.Ties++
			vr.Ties++
		} else if winnerSlot == 0 {
			or.Wins++
			vr.Losses++
		} else {
			or.Losses++
			vr.Wins++
		}

		if err := tx.Save(&or).Error; err != nil {
			return err
		}
		return tx.Save(&vr).Error
	})
}
