package baseline

import (
	"os"
	"strings"
	"testing"
)

// TestRestoreDeferredMarkerStructure pins the F5 contract: the tenant_baselines
// completion UPSERT (StatusComplete) runs only after every restoreOneTable
// iteration AND every ANALYZE succeeds, with cleanupAfterFailure invoked on any
// mid-restore error. Pre-fix per-table TXs could leave half-restored data while
// the marker UPSERT still ran, falsely advertising the restore as "ready."
func TestRestoreDeferredMarkerStructure(t *testing.T) {
	body := readRestoreSource(t)
	if !strings.Contains(body, "cleanupAfterFailure") {
		t.Fatal("restore.go missing cleanupAfterFailure helper required by F5")
	}
	// The completion marker is the finalize Exec, identified by its StatusComplete
	// argument (the earlier StatusRestoring intent write deliberately precedes the
	// loop, so a bare "INSERT INTO tenant_baselines" match is no longer unique).
	idxComplete := strings.Index(body, ", StatusComplete)")
	if idxComplete < 0 {
		t.Fatal("restore.go missing StatusComplete finalize UPSERT")
	}
	idxLoopEnd := strings.LastIndex(body, "restoreOneTable(")
	if idxLoopEnd < 0 {
		t.Fatal("restore.go missing restoreOneTable call")
	}
	if idxComplete < idxLoopEnd {
		t.Fatal("StatusComplete finalize appears before the restoreOneTable loop end; F5 requires deferred finalization")
	}
}

// TestRestoreIntentPrecedesTables pins that the StatusRestoring intent marker is
// written before the table COPY loop, so an interrupted restore leaves a durable
// record for Reconcile.
func TestRestoreIntentPrecedesTables(t *testing.T) {
	body := readRestoreSource(t)
	idxIntent := strings.Index(body, "r.markRestoring(ctx")
	if idxIntent < 0 {
		t.Fatal("restore.go missing markRestoring intent write")
	}
	idxLoop := strings.Index(body, "runRestoreTables(ctx, r.L")
	if idxLoop < 0 {
		t.Fatal("restore.go missing runRestoreTables call")
	}
	if idxIntent > idxLoop {
		t.Fatal("markRestoring must precede runRestoreTables so an interrupted restore is recoverable")
	}
}

// TestRestoreAcquiresTenantLockFirst pins the per-tenant advisory lock that
// serializes restores across replicas (Recreate rollout starts all replicas at
// once, each reconciling). The lock must be taken before any table COPY so two
// replicas can't interleave DELETE+COPY into one tenant.
func TestRestoreAcquiresTenantLockFirst(t *testing.T) {
	body := readRestoreSource(t)
	idxLock := strings.Index(body, "acquireTenantLock(ctx, r.DB, target)")
	if idxLock < 0 {
		t.Fatal("Restore must acquire a per-tenant lock via acquireTenantLock")
	}
	idxLoop := strings.Index(body, "runRestoreTables(ctx, r.L")
	if idxLoop < 0 || idxLock > idxLoop {
		t.Fatal("acquireTenantLock must be called before runRestoreTables")
	}
}

// TestRestoreContextDetached pins the two context invariants that turned a
// transient cancellation into the permanent atlas-pr-933 half-restore:
//  1. Restore detaches its DB/MinIO work from the request context
//     (context.WithoutCancel) so a proxy/client cancel can't abort a COPY.
//  2. cleanupAfterFailure runs under context.Background(), not the (possibly
//     cancelled) restore context, so the partial-state wipe actually executes.
func TestRestoreContextDetached(t *testing.T) {
	body := readRestoreSource(t)
	if !strings.Contains(body, "context.WithoutCancel(ctx)") {
		t.Fatal("Restore must detach from the request context via context.WithoutCancel")
	}
	idxCleanup := strings.Index(body, "func cleanupAfterFailure(")
	if idxCleanup < 0 {
		t.Fatal("restore.go missing cleanupAfterFailure definition")
	}
	// The next func after cleanupAfterFailure bounds its body.
	rest := body[idxCleanup+len("func cleanupAfterFailure("):]
	end := strings.Index(rest, "\nfunc ")
	if end >= 0 {
		rest = rest[:end]
	}
	if !strings.Contains(rest, "context.Background()") {
		t.Fatal("cleanupAfterFailure must derive its context from context.Background(), not the cancelled restore context")
	}
}

// TestRestoreTableSwapIsAtomic pins task-186: a table's DELETE and its
// repopulating COPY must run in ONE transaction on ONE connection, so a reader
// never observes the deleted-but-not-yet-repopulated gap — the transient
// GET /api/data/skills/{id} 404 seen during a baseline re-apply. The pre-fix bug
// ran the DELETE inside a gorm db.Transaction but had the COPY check out a
// SECOND pooled connection via tx.DB().Conn(ctx), so the COPY committed outside
// the DELETE's transaction (sql.DB.Conn never returns a transaction's own
// connection). This is a source-structure guard because the pgx binary-COPY path
// executes only against real Postgres, not the sqlite unit-test DB.
func TestRestoreTableSwapIsAtomic(t *testing.T) {
	body := readRestoreSource(t)
	const marker = "func replaceTableBinary("
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatal("restore.go missing replaceTableBinary (the atomic single-connection DELETE+COPY swap)")
	}
	fn := body[i:]
	if end := strings.Index(fn[len(marker):], "\nfunc "); end >= 0 {
		fn = fn[:len(marker)+end]
	}
	// DELETE + COPY bracketed by an explicit BEGIN/COMMIT on one connection.
	for _, need := range []string{"c.Begin(ctx)", "DELETE FROM ", "CopyFrom(", "tx.Commit(ctx)"} {
		if !strings.Contains(fn, need) {
			t.Fatalf("replaceTableBinary must contain %q so the DELETE and COPY form one committed transaction", need)
		}
	}
	// Guard against the pre-fix defect returning: the swap must NOT wrap the
	// DELETE in a gorm db.Transaction (whose COPY escaped onto a 2nd connection).
	if strings.Contains(fn, "db.Transaction(") {
		t.Fatal("replaceTableBinary must run DELETE+COPY on one raw connection, not a gorm db.Transaction")
	}
}

func readRestoreSource(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("restore.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}
