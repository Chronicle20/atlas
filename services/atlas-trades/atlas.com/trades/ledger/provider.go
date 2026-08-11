package ledger

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// withSides adds the tenant-scoped preloads for an Entry query. A preload
// issues its own SELECT, which the parent query's tenant filter does not reach,
// so both repeat it — a row written with a foreign tenant_id but a local
// entry_id/side_id would otherwise be handed back inside a local entry.
//
// Both preloads are explicitly ordered. Without an ORDER BY the row order is
// whatever the storage engine returns, and Model.Sides()[0] would be a
// coin flip; see the ordering note on Model.Sides.
func withSides(db *gorm.DB, tenantId uuid.UUID) *gorm.DB {
	return db.
		Preload("Sides", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("tenant_id = ?", tenantId).Order("character_id ASC")
		}).
		Preload("Sides.Items", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("tenant_id = ?", tenantId).Order("item_id ASC, id ASC")
		})
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

// sideExistsForCharacter is the correlated EXISTS that matches an entry on
// EITHER of its sides (FR-7.2). It carries its own tenant_id filter: the outer
// query's filter does not reach inside a subquery, and without it a character
// id that another tenant also uses would match this tenant's entries.
//
// EXISTS rather than collecting side ids and feeding them back as `id IN (…)`:
// that shape grows one bind parameter per matched trade and a busy character
// over a wide range would blow PostgreSQL's 65535-parameter limit.
var sideExistsForCharacter = "EXISTS (SELECT 1 FROM " + sideTable + " WHERE " +
	sideTable + ".entry_id = " + entryTable + ".id AND " +
	sideTable + ".tenant_id = ? AND " + sideTable + ".character_id = ?)"

// entriesByCharacterQuery is the shared FR-7.2 selection: this tenant's entries
// in [from, to] on which the character appears as either side, with both child
// preloads tenant-scoped. Every predicate carries tenant_id — the outer query,
// both preloads and the EXISTS subquery — because none of the three reaches the
// others.
func entriesByCharacterQuery(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, from time.Time, to time.Time) *gorm.DB {
	return withSides(db, tenantId).
		Where(entryTable+".tenant_id = ? AND "+entryTable+".settled_at >= ? AND "+entryTable+".settled_at <= ?", tenantId, from, to).
		Where(sideExistsForCharacter, tenantId, characterId)
}

// entriesByCharacterProvider yields every entry in [from, to] on which the
// character appears as either side (FR-7.2), newest first.
func entriesByCharacterProvider(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, from time.Time, to time.Time) model.Provider[[]Entry] {
	var es []Entry
	err := entriesByCharacterQuery(db, tenantId, characterId, from, to).
		Order("settled_at DESC").
		Find(&es).Error
	if err != nil {
		return model.ErrorProvider[[]Entry](err)
	}
	return model.FixedProvider(es)
}

// entriesByCharacterPagedProvider yields ONE PAGE of the same selection, paged
// in SQL rather than by materialising the character's whole history and slicing
// in Go (docs/rest-pagination.md §3: Entry has a single prioritized primary key,
// so it pages in the database).
//
// The settled_at ordering is applied here; database.PagedQuery appends the
// primary key as the tie-break, so two trades settled in the same instant still
// get a total order and a page boundary can neither drop nor repeat an entry.
func entriesByCharacterPagedProvider(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, from time.Time, to time.Time, page model.Page) model.Provider[model.Paged[Entry]] {
	return database.PagedQuery[Entry](
		entriesByCharacterQuery(db, tenantId, characterId, from, to).Order(entryTable+".settled_at DESC"),
		page,
	)
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

// pageByCharacter returns one page of the same selection, scoped to tenantId.
// It backs the REST list read, which must not materialise a busy character's
// whole trade history to serve 50 rows.
func pageByCharacter(db *gorm.DB, tenantId uuid.UUID) func(characterId character.Id, from time.Time, to time.Time, page model.Page) (model.Paged[Model], error) {
	return func(characterId character.Id, from time.Time, to time.Time, page model.Page) (model.Paged[Model], error) {
		return model.MapPaged(Make)(entriesByCharacterPagedProvider(db, tenantId, characterId, from, to, page))(model.ParallelMap())()
	}
}
