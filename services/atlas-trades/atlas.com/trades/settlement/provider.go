package settlement

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// withSides adds the tenant-scoped preloads for an Entry query. A preload
// issues its own SELECT, which the parent query's tenant filter does not reach,
// so both repeat it — a row written with a foreign tenant_id but a local
// entry_id/side_id would otherwise be handed back inside a local record.
//
// Sides are ordered by position so the pair is always owner-then-visitor, and
// items by asset id so a rebuilt record is byte-stable.
func withSides(db *gorm.DB, tenantId uuid.UUID) *gorm.DB {
	return db.
		Preload("Sides", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("tenant_id = ?", tenantId).Order("position ASC")
		}).
		Preload("Sides.Items", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("tenant_id = ?", tenantId).Order("asset_id ASC, id ASC")
		})
}

// byTransactionIdProvider yields the unresolved settlement with the given saga
// transaction id. (tenant_id, transaction_id) is unique, so this is at most one
// row.
func byTransactionIdProvider(db *gorm.DB, tenantId uuid.UUID, transactionId uuid.UUID) model.Provider[Entry] {
	var e Entry
	err := withSides(db, tenantId).
		Where("tenant_id = ? AND transaction_id = ?", tenantId, transactionId).
		First(&e).Error
	if err != nil {
		return model.ErrorProvider[Entry](err)
	}
	return model.FixedProvider(e)
}

func byTransactionId(db *gorm.DB, tenantId uuid.UUID) func(transactionId uuid.UUID) (Model, error) {
	return func(transactionId uuid.UUID) (Model, error) {
		return model.Map(Make)(byTransactionIdProvider(db, tenantId, transactionId))()
	}
}

// unresolvedForTenant yields the request tenant's unfinished settlements,
// oldest first.
func unresolvedForTenant(db *gorm.DB, tenantId uuid.UUID) ([]Model, error) {
	var entries []Entry
	err := withSides(db, tenantId).
		Where("tenant_id = ?", tenantId).
		Order("submitted_at ASC").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return model.SliceMap(Make)(model.FixedProvider(entries))(model.ParallelMap())()
}

// allUnresolved yields EVERY unfinished settlement across EVERY tenant, oldest
// first.
//
// It is deliberately un-scoped: startup reconciliation runs before any request
// has supplied a tenant, and each row carries the tenant fields needed to
// restore one (see Entry). Its preloads therefore cannot use the tenant-scoped
// withSides either — they are joined on the parent's id, and every row read
// here is already the parent of its own children.
//
// This is the ONLY query in the package that crosses tenants, and it is
// reachable only from the boot path.
func allUnresolved(db *gorm.DB) ([]Model, error) {
	var entries []Entry
	err := db.
		Preload("Sides", func(tx *gorm.DB) *gorm.DB { return tx.Order("position ASC") }).
		Preload("Sides.Items", func(tx *gorm.DB) *gorm.DB { return tx.Order("asset_id ASC, id ASC") }).
		Order("submitted_at ASC").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return model.SliceMap(Make)(model.FixedProvider(entries))(model.ParallelMap())()
}
