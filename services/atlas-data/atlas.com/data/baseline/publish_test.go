package baseline

import (
	"context"
	"strings"
	"testing"

	"atlas-data/canonical"

	"github.com/sirupsen/logrus"
)

// TestCopyOutSQLOrdersByTableKey is a regression test for the empty-500
// observed publishing v83/v84 baselines on atlas-main:
//
//	publish: dump-table monster_search_index: ERROR: column "id" does not exist (SQLSTATE 42703)
//
// runCopyOut hardcoded `ORDER BY id`, which only the documents table has.
// The *_search_index tables are keyed by (tenant_id, <entity>_id) with no
// surrogate `id`, so the COPY died on the second table and the dump+sidecar
// were never written. Every dumped table must order by a column it actually
// has.
func TestCopyOutSQLOrdersByTableKey(t *testing.T) {
	want := map[string]string{
		"documents":                "id",
		"monster_search_index":     "monster_id",
		"npc_search_index":         "npc_id",
		"reactor_search_index":     "reactor_id",
		"map_search_index":         "map_id",
		"item_string_search_index": "item_id",
	}
	for table, col := range want {
		sql := copyOutSQL(table, "GMS", 83, 1)
		// The trailing ")" closes the COPY subquery, pinning the exact column.
		if !strings.Contains(sql, "ORDER BY "+col+")") {
			t.Errorf("copyOutSQL(%q) = %q; want `ORDER BY %s)`", table, sql, col)
		}
	}
	// Guard: every table actually dumped must have a tested ordering, so a
	// future addition to DumpTables can't silently reintroduce `ORDER BY id`.
	for _, table := range DumpTables {
		if _, ok := want[table]; !ok {
			t.Errorf("DumpTables includes %q with no expected order column; add it here", table)
		}
	}
}

// TestCopyOutSQLUsesVersionScopedTenantId verifies that copyOutSQL filters on
// the version-derived canonical tenant UUID rather than the all-zeros sentinel.
// This is the core of T5: baseline publish must dump exactly the rows that were
// ingested under the version-scoped id, not the old sentinel.
func TestCopyOutSQLUsesVersionScopedTenantId(t *testing.T) {
	const region = "GMS"
	const major = 84
	const minor = 1

	expectedId := canonical.TenantId(region, uint16(major), uint16(minor)).String()
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	sql := copyOutSQL("documents", region, major, minor)

	if !strings.Contains(sql, "'"+expectedId+"'") {
		t.Errorf("copyOutSQL should contain version-scoped tenant id %q; got: %s", expectedId, sql)
	}
	if strings.Contains(sql, zeroUUID) {
		t.Errorf("copyOutSQL must not contain all-zeros sentinel %q; got: %s", zeroUUID, sql)
	}
}

// TestCopyOutSQLDistinctVersionsProduceDistinctIds verifies that different
// (region, major, minor) tuples produce different WHERE clauses — ensuring that
// a v83 publish and a v84 publish don't dump each other's rows.
func TestCopyOutSQLDistinctVersionsProduceDistinctIds(t *testing.T) {
	cases := []struct {
		region       string
		major, minor int
	}{
		{"GMS", 83, 1},
		{"GMS", 84, 1},
		{"GMS", 95, 1},
		{"JMS", 83, 1},
	}
	seen := make(map[string]struct{ region string; major, minor int })
	for _, c := range cases {
		sql := copyOutSQL("documents", c.region, c.major, c.minor)
		if prev, ok := seen[sql]; ok {
			t.Errorf("copyOutSQL(%q,%d,%d) == copyOutSQL(%q,%d,%d); version-scoped ids must differ",
				c.region, c.major, c.minor, prev.region, prev.major, prev.minor)
		}
		seen[sql] = c
	}
}

// TestPublishErrorIsContextualized asserts Publisher.Publish wraps every
// failure path with a `publish: <step>:` prefix so operators can locate the
// failing step in logs. Pre-fix Publisher returned raw errors (or empty
// io.Pipe failure modes), producing the empty-500 observed on atlas-main.
func TestPublishErrorIsContextualized(t *testing.T) {
	p := Publisher{DB: nil, MC: nil, L: logrus.New()}
	_, err := p.Publish(context.Background(), "GMS", 83, 1)
	if err == nil {
		t.Fatal("expected error from Publish with nil deps")
	}
	if !strings.Contains(err.Error(), "publish:") {
		t.Fatalf("error %q lacks `publish:` step prefix", err.Error())
	}
}
