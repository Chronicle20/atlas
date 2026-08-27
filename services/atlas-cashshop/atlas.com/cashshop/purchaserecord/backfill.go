package purchaserecord

import (
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// backfillGroupRow is one aggregated (tenant, account, commodity) purchase
// history group, as produced by the SQL-side GROUP BY/MIN/MAX aggregation.
type backfillGroupRow struct {
	TenantId    uuid.UUID
	AccountId   uint32
	CommodityId uint32
	Count       uint32
	FirstAt     time.Time
	LastAt      time.Time
}

// backfillAssetRow is one cash_assets row joined to its owning compartment's
// account id, ungrouped. Used only by the sqlite fallback grouping path --
// see backfillGroupsSQLite below.
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
//
// The idempotency gate below is a global count on cash_purchase_records, not
// per tenant, matching the table this backfill writes and Record (the live
// purchase path, wired since task-5) both write to. This assumes task-5 and
// task-6 ship together: if a live purchase is ever recorded before this
// backfill's first successful run, the backfill becomes a permanent no-op
// for every other tenant on every later boot. That is an acceptable
// assumption for this rollout (same branch, same release) but would need
// per-tenant scoping if task-6 were ever deployed after task-5 with a gap.
func Backfill(l logrus.FieldLogger, db *gorm.DB) (int, error) {
	var existing int64
	if err := db.Model(&entity{}).Count(&existing).Error; err != nil {
		return 0, err
	}
	if existing > 0 {
		l.Debugf("purchaserecord: %d record(s) already present, skipping backfill.", existing)
		return 0, nil
	}

	var groups []backfillGroupRow
	var err error
	if db.Name() == "sqlite" {
		// gorm's sqlite driver (mattn/go-sqlite3) cannot scan a MIN()/MAX()
		// over a DATETIME column back into time.Time -- the aggregate result
		// carries no declared column type for the driver's automatic
		// conversion to key off ("unsupported Scan, storing driver.Value
		// type string into type *time.Time"). This only affects the sqlite
		// test driver; production runs Postgres, which handles the SQL-side
		// aggregation below directly. Route sqlite through an equivalent
		// Go-side grouping instead of loading it into the boot-time SQL
		// aggregate path.
		groups, err = backfillGroupsSQLite(db)
	} else {
		groups, err = backfillGroupsSQL(db)
	}
	if err != nil {
		return 0, err
	}

	if len(groups) == 0 {
		l.Debugf("purchaserecord: no recoverable cash_assets history found. Note: assets that have already been hard-deleted are unrecoverable.")
		return 0, nil
	}

	entities := make([]entity, 0, len(groups))
	for _, g := range groups {
		entities = append(entities, entity{
			Id:           uuid.New(),
			TenantId:     g.TenantId,
			AccountId:    g.AccountId,
			SerialNumber: g.CommodityId,
			Count:        g.Count,
			FirstAt:      g.FirstAt,
			LastAt:       g.LastAt,
		})
	}

	if err := db.Create(&entities).Error; err != nil {
		return 0, err
	}

	l.Infof("purchaserecord: backfilled %d purchase record(s) from existing cash_assets history. Assets already hard-deleted before this backfill ran are unrecoverable.", len(entities))
	return len(entities), nil
}

// backfillGroupsSQL performs the grouping and aggregation entirely in SQL, in
// a single query bounded to the number of distinct (tenant, account,
// commodity) groups rather than the full row count of cash_assets. This is
// the production path (Postgres and any other driver whose MIN()/MAX()
// scans a DATETIME aggregate back into time.Time correctly).
func backfillGroupsSQL(db *gorm.DB) ([]backfillGroupRow, error) {
	var groups []backfillGroupRow
	err := db.Unscoped().
		Table("cash_assets").
		Select("cash_assets.tenant_id AS tenant_id, cash_compartments.account_id AS account_id, cash_assets.commodity_id AS commodity_id, COUNT(*) AS count, MIN(cash_assets.created_at) AS first_at, MAX(cash_assets.created_at) AS last_at").
		Joins("JOIN cash_compartments ON cash_compartments.id = cash_assets.compartment_id").
		Where("cash_assets.commodity_id <> 0").
		Group("cash_assets.tenant_id, cash_compartments.account_id, cash_assets.commodity_id").
		Scan(&groups).Error
	return groups, err
}

// backfillGroupsSQLite is the sqlite fallback: it reads the ungrouped join in
// one query and aggregates in Go, because sqlite cannot scan a MIN()/MAX()
// over a DATETIME column into time.Time (see the comment on the dialect
// check in Backfill). This path is only exercised by the module's sqlite
// test suite -- production runs Postgres via backfillGroupsSQL above, so an
// unbounded row count here is a test-fixture concern, not a production one.
func backfillGroupsSQLite(db *gorm.DB) ([]backfillGroupRow, error) {
	var assetRows []backfillAssetRow
	err := db.Unscoped().
		Table("cash_assets").
		Select("cash_assets.tenant_id AS tenant_id, cash_compartments.account_id AS account_id, cash_assets.commodity_id AS commodity_id, cash_assets.created_at AS created_at").
		Joins("JOIN cash_compartments ON cash_compartments.id = cash_assets.compartment_id").
		Where("cash_assets.commodity_id <> 0").
		Scan(&assetRows).Error
	if err != nil {
		return nil, err
	}

	type accum struct {
		count   uint32
		firstAt time.Time
		lastAt  time.Time
	}
	byKey := make(map[backfillGroupKey]*accum, len(assetRows))
	order := make([]backfillGroupKey, 0, len(assetRows))
	for _, r := range assetRows {
		key := backfillGroupKey{TenantId: r.TenantId, AccountId: r.AccountId, CommodityId: r.CommodityId}
		a, ok := byKey[key]
		if !ok {
			a = &accum{count: 0, firstAt: r.CreatedAt, lastAt: r.CreatedAt}
			byKey[key] = a
			order = append(order, key)
		}
		a.count++
		if r.CreatedAt.Before(a.firstAt) {
			a.firstAt = r.CreatedAt
		}
		if r.CreatedAt.After(a.lastAt) {
			a.lastAt = r.CreatedAt
		}
	}

	groups := make([]backfillGroupRow, 0, len(order))
	for _, key := range order {
		a := byKey[key]
		groups = append(groups, backfillGroupRow{
			TenantId:    key.TenantId,
			AccountId:   key.AccountId,
			CommodityId: key.CommodityId,
			Count:       a.count,
			FirstAt:     a.firstAt,
			LastAt:      a.lastAt,
		})
	}
	return groups, nil
}
