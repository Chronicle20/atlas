package ledger

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// withSides adds the tenant-scoped preloads for an Entry query. A preload
// issues its own SELECT, which the parent query's tenant filter does not reach,
// so both repeat it. The foreign keys already confine a child to its own
// tenant's parent, making this defence in depth rather than the only guard.
func withSides(db *gorm.DB, tenantId uuid.UUID) *gorm.DB {
	return db.
		Preload("Sides", "tenant_id = ?", tenantId).
		Preload("Sides.Items", "tenant_id = ?", tenantId)
}

// entryByIdProvider yields the entry with the given id in the given tenant.
func entryByIdProvider(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) model.Provider[Entry] {
	var e Entry
	err := withSides(db, tenantId).
		Where("tenant_id = ? AND id = ?", tenantId, id).
		First(&e).Error
	if err != nil {
		return model.ErrorProvider[Entry](err)
	}
	return model.FixedProvider(e)
}

// entryByTransactionIdProvider yields the entry recorded for the given
// settlement transaction. (tenant_id, transaction_id) is unique, so this is at
// most one row — it is how the idempotency guard reads back an already
// recorded settlement (FR-5.7).
func entryByTransactionIdProvider(db *gorm.DB, tenantId uuid.UUID, transactionId uuid.UUID) model.Provider[Entry] {
	var e Entry
	err := withSides(db, tenantId).
		Where("tenant_id = ? AND transaction_id = ?", tenantId, transactionId).
		First(&e).Error
	if err != nil {
		return model.ErrorProvider[Entry](err)
	}
	return model.FixedProvider(e)
}

// entriesByCharacterProvider yields every entry in [from, to] on which the
// character appears as either side (FR-7.2), newest first. The side lookup is
// a separate tenant-scoped query rather than a join so that neither half can
// silently lose its tenant filter.
func entriesByCharacterProvider(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, from time.Time, to time.Time) model.Provider[[]Entry] {
	var entryIds []uuid.UUID
	err := db.Model(&Side{}).
		Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
		Pluck("entry_id", &entryIds).Error
	if err != nil {
		return model.ErrorProvider[[]Entry](err)
	}
	if len(entryIds) == 0 {
		return model.FixedProvider([]Entry{})
	}

	var es []Entry
	err = withSides(db, tenantId).
		Where("tenant_id = ? AND id IN ? AND settled_at >= ? AND settled_at <= ?", tenantId, entryIds, from, to).
		Order("settled_at DESC").
		Find(&es).Error
	if err != nil {
		return model.ErrorProvider[[]Entry](err)
	}
	return model.FixedProvider(es)
}

// byId returns the ledger entry with the given id, scoped to tenantId.
func byId(db *gorm.DB, tenantId uuid.UUID) func(id uuid.UUID) (Model, error) {
	return func(id uuid.UUID) (Model, error) {
		e, err := entryByIdProvider(db, tenantId, id)()
		if err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

// byTransactionId returns the ledger entry recorded for the given settlement
// transaction, scoped to tenantId.
func byTransactionId(db *gorm.DB, tenantId uuid.UUID) func(transactionId uuid.UUID) (Model, error) {
	return func(transactionId uuid.UUID) (Model, error) {
		e, err := entryByTransactionIdProvider(db, tenantId, transactionId)()
		if err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

// byCharacter returns every ledger entry in [from, to] on which the character
// appears as either side, scoped to tenantId.
func byCharacter(db *gorm.DB, tenantId uuid.UUID) func(characterId character.Id, from time.Time, to time.Time) ([]Model, error) {
	return func(characterId character.Id, from time.Time, to time.Time) ([]Model, error) {
		return model.SliceMap(Make)(entriesByCharacterProvider(db, tenantId, characterId, from, to))()()
	}
}
