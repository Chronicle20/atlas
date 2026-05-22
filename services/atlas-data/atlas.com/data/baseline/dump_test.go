package baseline

import "testing"

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
