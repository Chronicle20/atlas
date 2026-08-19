package purchaserecord

import (
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// backfillAssetRow is one cash_assets row joined to its owning compartment's
// account id, ungrouped -- grouping happens in Go, below, because sqlite's
// driver cannot scan a MIN()/MAX() over a DATETIME column back into
// time.Time (the aggregate result carries no declared column type for the
// driver's automatic conversion to key off).
type backfillAssetRow struct {
	TenantId    uuid.UUID
	AccountId   uint32
	CommodityId uint32
	CreatedAt   time.Time
}

// backfillGroupKey identifies one (tenant, account, commodity) purchase
// history group.
type backfillGroupKey struct {
	TenantId    uuid.UUID
	AccountId   uint32
	CommodityId uint32
}

// Backfill seeds cash_purchase_records from existing cash_assets rows, once.
// It is a no-op on a database that already has records, so it is safe to run
// on every boot. Returns the number of records written.
//
// It runs unscoped (no tenant filter, no soft-delete filter) because it must
// see every tenant and, critically, assets that were soft-deleted by
// withdrawal or rebate -- that history is exactly what this backfill exists
// to recover. Assets that have been hard-deleted are gone for good; there is
// nothing left in cash_assets to recover them from.
func Backfill(l logrus.FieldLogger, db *gorm.DB) (int, error) {
	var existing int64
	if err := db.Model(&entity{}).Count(&existing).Error; err != nil {
		return 0, err
	}
	if existing > 0 {
		l.Debugf("purchaserecord: %d record(s) already present, skipping backfill.", existing)
		return 0, nil
	}

	var assetRows []backfillAssetRow
	err := db.Unscoped().
		Table("cash_assets").
		Select("cash_assets.tenant_id AS tenant_id, cash_compartments.account_id AS account_id, cash_assets.commodity_id AS commodity_id, cash_assets.created_at AS created_at").
		Joins("JOIN cash_compartments ON cash_compartments.id = cash_assets.compartment_id").
		Where("cash_assets.commodity_id <> 0").
		Scan(&assetRows).Error
	if err != nil {
		return 0, err
	}

	if len(assetRows) == 0 {
		l.Debugf("purchaserecord: no recoverable cash_assets history found. Note: assets that have already been hard-deleted are unrecoverable.")
		return 0, nil
	}

	type group struct {
		count   uint32
		firstAt time.Time
		lastAt  time.Time
	}
	groups := make(map[backfillGroupKey]*group, len(assetRows))
	order := make([]backfillGroupKey, 0, len(assetRows))
	for _, r := range assetRows {
		key := backfillGroupKey{TenantId: r.TenantId, AccountId: r.AccountId, CommodityId: r.CommodityId}
		g, ok := groups[key]
		if !ok {
			g = &group{count: 0, firstAt: r.CreatedAt, lastAt: r.CreatedAt}
			groups[key] = g
			order = append(order, key)
		}
		g.count++
		if r.CreatedAt.Before(g.firstAt) {
			g.firstAt = r.CreatedAt
		}
		if r.CreatedAt.After(g.lastAt) {
			g.lastAt = r.CreatedAt
		}
	}

	entities := make([]entity, 0, len(order))
	for _, key := range order {
		g := groups[key]
		entities = append(entities, entity{
			Id:           uuid.New(),
			TenantId:     key.TenantId,
			AccountId:    key.AccountId,
			SerialNumber: key.CommodityId,
			Count:        g.count,
			FirstAt:      g.firstAt,
			LastAt:       g.lastAt,
		})
	}

	if err := db.Create(&entities).Error; err != nil {
		return 0, err
	}

	l.Infof("purchaserecord: backfilled %d purchase record(s) from existing cash_assets history. Assets already hard-deleted before this backfill ran are unrecoverable.", len(entities))
	return len(entities), nil
}
