package coupon_test

// NOTE: This test runs against gorm's SQLite in-memory driver via
// databasetest.NewInMemoryTenantDB, NOT Postgres. A human ruling on this
// branch selected SQLite in-memory as the harness for this plan's DB tests
// (testcontainers Postgres was available and deliberately declined). SQLite's
// index machinery is different from Postgres', so passing here proves the
// GORM struct tags on Entity produce the intended indexes — one composite
// UNIQUE index per table, on the expected column set, in the expected
// priority order — but it does NOT prove Postgres itself enforces uniqueness
// the same way at the storage-engine/MVCC level. That is what it means to say
// this validates the tags, not Postgres semantics.

import (
	"atlas-cashshop/coupon"
	"atlas-cashshop/coupon/batch"
	"atlas-cashshop/coupon/redemption"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func TestMigration_UniqueIndexes(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, coupon.Migration, batch.Migration, redemption.Migration)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// coupons: uniqueIndex on (tenant_id, code) — idx_coupons_tenant_code.
	assertUniqueIndex(t, sqlDB, "coupons", "idx_coupons_tenant_code", []string{"tenant_id", "code"})

	// coupon_redemptions: uniqueIndex on (tenant_id, coupon_id, account_id) —
	// this is the one-time-per-account rule (design §2.2). A non-unique index
	// here would silently disable it, so this assertion is the point of the
	// test, not a side effect of it.
	assertUniqueIndex(t, sqlDB, "coupon_redemptions", "idx_redemptions_tenant_coupon_account", []string{"tenant_id", "coupon_id", "account_id"})
}

// assertUniqueIndex asserts that table carries an index named indexName,
// that SQLite's `PRAGMA index_list(table)` reports it as unique (the `unique`
// column = 1), and that `PRAGMA index_info(indexName)` reports exactly
// wantCols, in that priority order.
func assertUniqueIndex(t *testing.T, db *sql.DB, table, indexName string, wantCols []string) {
	t.Helper()

	rows, err := db.Query("PRAGMA index_list(" + table + ")")
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	found := false
	unique := false
	for rows.Next() {
		var seq int
		var name, origin string
		var uniq, partial int
		require.NoError(t, rows.Scan(&seq, &name, &uniq, &origin, &partial))
		if name == indexName {
			found = true
			unique = uniq == 1
		}
	}
	require.NoError(t, rows.Err())
	require.True(t, found, "expected index %q on table %q to exist", indexName, table)
	require.True(t, unique, "expected index %q on table %q to be UNIQUE", indexName, table)

	colRows, err := db.Query("PRAGMA index_info(" + indexName + ")")
	require.NoError(t, err)
	defer func() { _ = colRows.Close() }()

	var gotCols []string
	for colRows.Next() {
		var seqno, cid int
		var colName string
		require.NoError(t, colRows.Scan(&seqno, &cid, &colName))
		gotCols = append(gotCols, colName)
	}
	require.NoError(t, colRows.Err())
	require.Equal(t, wantCols, gotCols, "index %q column set/order on table %q", indexName, table)
}
