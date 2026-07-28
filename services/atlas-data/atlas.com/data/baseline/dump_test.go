package baseline

import (
	"regexp"
	"strings"
	"testing"
)

func TestDumpKey(t *testing.T) {
	if got := DumpKey("GMS", 83, 1); got != "baseline/regions/GMS/versions/83.1/documents.dump" {
		t.Fatalf("DumpKey = %s", got)
	}
}

func TestShaKey(t *testing.T) {
	if got := ShaKey("GMS", 83, 1); got != "baseline/regions/GMS/versions/83.1/documents.dump.sha256" {
		t.Fatalf("ShaKey = %s", got)
	}
}

func TestDumpTablesContainsDocuments(t *testing.T) {
	found := false
	for _, tbl := range DumpTables {
		if tbl == "documents" {
			found = true
		}
	}
	if !found {
		t.Fatal("documents missing from DumpTables")
	}
}

// typeColumnRef matches a bare `type` identifier reference (optionally
// double-quoted, any spacing/operator around it) — used to detect a `type`
// predicate in a WHERE clause regardless of how it's spelled (`type =`,
// `type='x'`, `type IN (...)`, `type <> 'x'`, `"type" = `, ...).
var typeColumnRef = regexp.MustCompile(`\btype\b`)

// TestCopyOutSQLDocumentsHasNoTypeFilter pins design D9 / FR-1.4: the
// documents dump's WHERE clause references no `type` column at all — it is
// whole-table, filtered on tenant only. A future per-`type` predicate, in any
// spelling, would silently drop JOB rows (and every other document type added
// after this test was written) from every published baseline.
func TestCopyOutSQLDocumentsHasNoTypeFilter(t *testing.T) {
	sql := copyOutSQL("documents", []string{"tenant_id", "type", "document_id", "content"}, "GMS", 83, 1)
	whereIdx := strings.Index(sql, "WHERE")
	if whereIdx == -1 {
		t.Fatalf("documents dump has no WHERE clause: %s", sql)
	}
	// Scope the check to the WHERE clause only: the SELECT column list
	// legitimately contains "type" (it's a real column being dumped), and
	// that must not trip this assertion.
	where := sql[whereIdx:]
	if typeColumnRef.MatchString(where) {
		t.Fatalf("documents dump's WHERE clause references `type`; a per-type filter would silently drop JOB rows: %s", sql)
	}
	if !strings.Contains(sql, "FROM documents WHERE tenant_id = ") {
		t.Fatalf("documents dump is no longer tenant-filtered whole-table: %s", sql)
	}
}
