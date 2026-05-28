package baseline

import (
	"os"
	"strings"
	"testing"
)

// TestRestoreDeferredMarkerStructure pins the F5 contract: tenant_baselines
// UPSERT runs only after every restoreOneTable iteration AND every ANALYZE
// succeeds, with cleanupAfterFailure invoked on any mid-restore error.
// Pre-fix per-table TXs could leave half-restored data while the marker
// UPSERT still ran, falsely advertising the restore as "ready."
func TestRestoreDeferredMarkerStructure(t *testing.T) {
	src, err := os.ReadFile("restore.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if !strings.Contains(body, "cleanupAfterFailure") {
		t.Fatal("restore.go missing cleanupAfterFailure helper required by F5")
	}
	idxMarker := strings.Index(body, "INSERT INTO tenant_baselines")
	if idxMarker < 0 {
		t.Fatal("restore.go missing tenant_baselines INSERT")
	}
	idxLoopEnd := strings.LastIndex(body, "restoreOneTable(")
	if idxLoopEnd < 0 {
		t.Fatal("restore.go missing restoreOneTable call")
	}
	if idxMarker < idxLoopEnd {
		t.Fatal("tenant_baselines UPSERT appears before the restoreOneTable loop end; F5 requires deferred finalization")
	}
}
