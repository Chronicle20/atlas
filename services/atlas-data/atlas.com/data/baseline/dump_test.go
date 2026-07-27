package baseline

import (
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

// TestCopyOutSQLDocumentsHasNoTypeFilter pins design D9 / FR-1.4: the documents
// dump is whole-table, filtered on tenant only. A future per-`type` filter
// would silently drop JOB rows (and every other document type added after this
// test was written) from every published baseline.
func TestCopyOutSQLDocumentsHasNoTypeFilter(t *testing.T) {
	sql := copyOutSQL("documents", []string{"tenant_id", "type", "document_id", "content"}, "GMS", 83, 1)
	if strings.Contains(sql, "type =") || strings.Contains(sql, `"type" =`) {
		t.Fatalf("documents dump gained a type filter; JOB rows would be dropped: %s", sql)
	}
	if !strings.Contains(sql, "FROM documents WHERE tenant_id = ") {
		t.Fatalf("documents dump is no longer tenant-filtered whole-table: %s", sql)
	}
}
