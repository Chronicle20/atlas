package baseline

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// ErrSchemaMismatch indicates the dump header schemaVersion does not match the
// service's current SchemaVersion constant. Surfaced as 422 by the handler.
var ErrSchemaMismatch = errors.New("dump schema mismatch")

// ErrShaMismatch indicates the computed sha256 of the tar stream did not match
// the sha256 sidecar object. Surfaced as 422 by the handler.
var ErrShaMismatch = errors.New("sha256 mismatch")

// ErrRestoreInProgress indicates another restore already holds the per-tenant
// advisory lock. Surfaced as 409 by the handler; Reconcile treats it as a
// benign skip. It exists because atlas-data runs multiple replicas with a
// Recreate strategy — a rolling restart starts every replica together, and
// each would otherwise run Reconcile and race Restore() for the same tenant,
// interleaving DELETE+COPY across the shared DumpTables.
var ErrRestoreInProgress = errors.New("restore already in progress for tenant")

// restoreOpTimeout bounds a single restore's DB/MinIO work once it is detached
// from the caller's request context. A full canonical restore re-COPYs every
// DumpTable (documents is ~50k rows) and can run several minutes under shared-
// Postgres contention; 30 min is a generous ceiling that still kills a wedged
// restore. cleanupTimeout bounds the best-effort partial-state wipe.
const (
	restoreOpTimeout = 30 * time.Minute
	cleanupTimeout   = 2 * time.Minute
)

// Restorer applies a canonical baseline dump to a single target tenant.
type Restorer struct {
	DB *gorm.DB
	MC *minio.Client
	L  logrus.FieldLogger
}

// runRestoreTables consumes the tar reader and dispatches each table entry
// to restoreOneTable. Pulled out of Restore so the marker UPSERT can be
// deferred until every entry succeeds. columns is the header's per-table
// column list (the order the dump's COPY stream was produced with).
func runRestoreTables(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, tr *tar.Reader, target uuid.UUID, columns map[string][]string) error {
	for {
		e, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		table := strings.TrimSuffix(e.Name, ".binary")
		if !contains(DumpTables, table) {
			return fmt.Errorf("unexpected table %s", table)
		}
		cols := columns[table]
		if len(cols) == 0 {
			return fmt.Errorf("restore table %s: header has no column list (re-publish with schema %s)", table, SchemaVersion)
		}
		if err := restoreOneTable(ctx, l, db, table, cols, tr, target); err != nil {
			return fmt.Errorf("restore table %s: %w", table, err)
		}
	}
}

func restoreOneTable(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, table string, cols []string, r io.Reader, target uuid.UUID) error {
	tenantIdx := columnIndex(cols, "tenant_id")
	if tenantIdx < 0 {
		return fmt.Errorf("column list has no tenant_id")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", target.String()).Error; err != nil {
			return err
		}
		rw := Rewriter{TenantColIndex: tenantIdx, Target: target}
		// Pipe rw.Stream() into COPY <table> (cols) FROM STDIN BINARY through
		// the raw connection. The explicit column list maps stream fields to
		// columns by NAME, so the target's physical column order is irrelevant.
		return copyInBinary(ctx, l, tx, table, cols, r, rw)
	})
}

// cleanupAfterFailure DELETEs every DumpTables row for target in its own
// best-effort transaction so a subsequent restore is not blocked by stale
// rows. Cleanup failures are logged; the caller still returns the original
// restore error so operators see the true cause.
//
// The cleanup context is derived from context.Background(), deliberately NOT
// from the restore's context. Cleanup must run even when the restore failed
// *because* its context was cancelled — reusing that cancelled context made
// every DELETE no-op with "context canceled", leaving item_string_search_index
// (the last, largest table) empty and the tenant permanently half-restored
// (atlas-pr-933). A fresh bounded context guarantees the wipe executes so a
// failed restore reads as "never restored" rather than half-restored.
func cleanupAfterFailure(l logrus.FieldLogger, db *gorm.DB, target uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	for _, t := range DumpTables {
		if err := db.WithContext(ctx).Exec("DELETE FROM "+t+" WHERE tenant_id = ?", target.String()).Error; err != nil {
			l.WithError(err).Warnf("restore: cleanup DELETE FROM %s failed (best-effort)", t)
		}
	}
}

// Restore is destructive: DELETE rows for target tenant, COPY-FROM with rewrite,
// ANALYZE, UPSERT tenant_baselines.
//
// The sha256 of the downloaded dump is verified against the sidecar BEFORE any
// DB mutation. The dump is staged to a temp file so the tar reader can be
// rewound after hashing without re-downloading.
func (r Restorer) Restore(ctx context.Context, region string, major, minor int, target uuid.UUID) error {
	// Detach from the caller's request context. A 30s ingress/proxy deadline or
	// a client disconnect must NOT abort a multi-minute restore mid-COPY — that
	// is exactly what left item_string_search_index (the last, largest table)
	// empty in atlas-pr-933. WithoutCancel keeps the tenant/trace values while
	// dropping cancellation; the own timeout still bounds a wedged restore.
	opCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restoreOpTimeout)
	defer cancel()
	ctx = opCtx

	// Serialize restores per tenant across all atlas-data replicas: a Recreate
	// rollout brings every replica up together and each runs Reconcile, so
	// without this lock 4+ pods would race Restore() for the same tenant. The
	// lock auto-releases if the holding pod dies (session-scoped), so a crash
	// mid-restore leaves the StatusRestoring marker for the next Reconcile.
	release, ok, err := acquireTenantLock(ctx, r.DB, target)
	if err != nil {
		return err
	}
	if !ok {
		r.L.Infof("restore: tenant=%s already being restored elsewhere; skipping", target)
		return ErrRestoreInProgress
	}
	defer release()

	sumBytes, err := readMinioObject(ctx, r.MC, r.MC.Cfg().BucketCanonical, ShaKey(region, major, minor))
	if err != nil {
		return err
	}
	expectedSum := strings.TrimSpace(string(sumBytes))

	// 1) Download and hash BEFORE any mutation.
	dumpRC, err := r.MC.Get(ctx, r.MC.Cfg().BucketCanonical, DumpKey(region, major, minor))
	if err != nil {
		return err
	}
	defer dumpRC.Close()

	tmp, err := os.CreateTemp("", "baseline-*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), dumpRC); err != nil {
		return err
	}
	actualSum := hex.EncodeToString(h.Sum(nil))
	if actualSum != expectedSum {
		return fmt.Errorf("%w: expected=%s got=%s", ErrShaMismatch, expectedSum, actualSum)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		return err
	}

	// 2) Validate header.
	tr := tar.NewReader(tmp)
	hdrEntry, err := tr.Next()
	if err != nil {
		return err
	}
	if hdrEntry.Name != "header.json" {
		return fmt.Errorf("expected header.json, got %s", hdrEntry.Name)
	}
	var hdr Header
	if err := json.NewDecoder(tr).Decode(&hdr); err != nil {
		return err
	}
	if hdr.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: dump=%s current=%s", ErrSchemaMismatch, hdr.SchemaVersion, SchemaVersion)
	}

	// 3) Persist a StatusRestoring marker BEFORE any table COPY, autocommitted
	//    in its own statement so it survives a later rollback/crash. region,
	//    major and minor live here because the documents table carries no
	//    version — this row is how startup reconciliation knows which baseline
	//    to re-run for an interrupted restore.
	if err := r.markRestoring(ctx, region, major, minor, target, expectedSum); err != nil {
		return err
	}

	// 4) Mutations only after both gates pass. Two-phase finalization: per-
	//    table transactions are unchanged; the tenant_baselines completion
	//    UPSERT (StatusComplete) is deferred until every table loop iteration
	//    AND every ANALYZE succeeds. A mid-restore failure triggers
	//    cleanupAfterFailure to DELETE every DumpTables row for `target` so
	//    subsequent reads see "never restored" rather than half-restored; the
	//    StatusRestoring marker is left in place (it is not a DumpTable) so
	//    Reconcile re-runs the restore on the next startup. See task-076 F5.
	if err := runRestoreTables(ctx, r.L, r.DB, tr, target, hdr.Columns); err != nil {
		r.L.WithError(err).Warnf("restore: table loop failed for target=%s region=%s ver=%d.%d; cleaning partial state", target, region, major, minor)
		cleanupAfterFailure(r.L, r.DB, target)
		return err
	}

	for _, t := range DumpTables {
		if err := r.DB.WithContext(ctx).Exec("ANALYZE " + t).Error; err != nil {
			r.L.WithError(err).Warnf("restore: ANALYZE %s failed; cleaning partial state", t)
			cleanupAfterFailure(r.L, r.DB, target)
			return err
		}
	}

	if err := r.DB.WithContext(ctx).Exec(`
        INSERT INTO tenant_baselines (tenant_id, region, major_version, minor_version, baseline_sha256, restored_at, status)
        VALUES (?, ?, ?, ?, ?, now(), ?)
        ON CONFLICT (tenant_id) DO UPDATE SET region=EXCLUDED.region, major_version=EXCLUDED.major_version,
            minor_version=EXCLUDED.minor_version, baseline_sha256=EXCLUDED.baseline_sha256, restored_at=now(), status=EXCLUDED.status
    `, target.String(), region, major, minor, expectedSum, StatusComplete).Error; err != nil {
		return err
	}
	r.L.Infof("restore: finalized target=%s region=%s ver=%d.%d sha=%s", target, region, major, minor, expectedSum)
	return nil
}

// markRestoring UPSERTs the tenant_baselines row as StatusRestoring before the
// first table COPY, autocommitted (not inside the per-table transactions) so an
// interruption leaves a durable "restore in progress" record for Reconcile.
func (r Restorer) markRestoring(ctx context.Context, region string, major, minor int, target uuid.UUID, sha string) error {
	return r.DB.WithContext(ctx).Exec(`
        INSERT INTO tenant_baselines (tenant_id, region, major_version, minor_version, baseline_sha256, restored_at, status)
        VALUES (?, ?, ?, ?, ?, now(), ?)
        ON CONFLICT (tenant_id) DO UPDATE SET region=EXCLUDED.region, major_version=EXCLUDED.major_version,
            minor_version=EXCLUDED.minor_version, baseline_sha256=EXCLUDED.baseline_sha256, status=EXCLUDED.status
    `, target.String(), region, major, minor, sha, StatusRestoring).Error
}

// advisoryKey derives a stable int64 advisory-lock key from a tenant UUID.
func advisoryKey(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[:8]))
}

// acquireTenantLock takes a Postgres session-level advisory lock keyed by the
// tenant on a dedicated connection held for the restore's lifetime, so at most
// one restore runs per tenant across all replicas. ok=false means another
// restore already holds it. The returned release MUST be called: pg_advisory_
// unlock runs on a non-cancellable context (the restore's own ctx may already
// be expired) before the connection returns to the pool, since a session-level
// lock would otherwise leak onto the pooled connection. On a non-Postgres
// dialect (unit tests) locking is skipped and a no-op release is returned.
func acquireTenantLock(ctx context.Context, db *gorm.DB, target uuid.UUID) (release func(), ok bool, err error) {
	if db.Dialector.Name() != "postgres" {
		return func() {}, true, nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, false, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	key := advisoryKey(target)
	var got bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&got); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !got {
		_ = conn.Close()
		return nil, false, nil
	}
	return func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", key)
		_ = conn.Close()
	}, true, nil
}

// columnIndex returns the position of name in cols, or -1. Used to locate the
// tenant_id field in the dump's recorded column list so the Rewriter rewrites
// the right field regardless of where tenant_id sits per table.
func columnIndex(cols []string, name string) int {
	for i, c := range cols {
		if c == name {
			return i
		}
	}
	return -1
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func readMinioObject(ctx context.Context, mc *minio.Client, bucket, key string) ([]byte, error) {
	rc, err := mc.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// copyInBinary pipes rw.Stream(in,…) into
// `COPY <table> (cols) FROM STDIN (FORMAT binary)` through the underlying pgx
// connection borrowed from the gorm transaction. The explicit column list
// mirrors the publish-time projection so Postgres maps stream fields to columns
// by name, not by the target table's physical order.
func copyInBinary(ctx context.Context, l logrus.FieldLogger, tx *gorm.DB, table string, cols []string, in io.Reader, rw Rewriter) error {
	sqlDB, err := tx.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		pgxConn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("expected *stdlib.Conn, got %T", driverConn)
		}
		pr, pw := io.Pipe()
		errc := make(chan error, 1)
		routine.Go(l, ctx, func(_ context.Context) {
			defer pw.Close()
			errc <- rw.Stream(in, pw)
		})
		sql := fmt.Sprintf(`COPY %s (%s) FROM STDIN (FORMAT binary)`, table, quoteCols(cols))
		if _, err := pgxConn.Conn().PgConn().CopyFrom(ctx, pr, sql); err != nil {
			// Drain the rewriter goroutine so it doesn't deadlock writing to pw.
			_ = pr.CloseWithError(err)
			<-errc
			return err
		}
		return <-errc
	})
}
