package baseline

import (
	"context"
	"strings"
	"testing"

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
		sql := copyOutSQL(table)
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
