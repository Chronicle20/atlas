package baseline

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	minio "atlas-data/storage/minio"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ErrSchemaMismatch indicates the dump header schemaVersion does not match the
// service's current SchemaVersion constant. Surfaced as 422 by the handler.
var ErrSchemaMismatch = errors.New("dump schema mismatch")

// ErrShaMismatch indicates the computed sha256 of the tar stream did not match
// the sha256 sidecar object. Surfaced as 422 by the handler.
var ErrShaMismatch = errors.New("sha256 mismatch")

// Restorer applies a canonical baseline dump to a single target tenant.
type Restorer struct {
	DB *gorm.DB
	MC *minio.Client
	L  logrus.FieldLogger
}

// runRestoreTables consumes the tar reader and dispatches each table entry
// to restoreOneTable. Pulled out of Restore so the marker UPSERT can be
// deferred until every entry succeeds.
func runRestoreTables(ctx context.Context, db *gorm.DB, tr *tar.Reader, target uuid.UUID) error {
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
		if err := restoreOneTable(ctx, db, table, tr, target); err != nil {
			return fmt.Errorf("restore table %s: %w", table, err)
		}
	}
}

func restoreOneTable(ctx context.Context, db *gorm.DB, table string, r io.Reader, target uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", target.String()).Error; err != nil {
			return err
		}
		rw := Rewriter{TenantColIndex: tenantColIndex(table), Target: target}
		// Pipe rw.Stream() into COPY FROM STDIN BINARY through the raw connection.
		return copyInBinary(ctx, tx, table, r, rw)
	})
}

// cleanupAfterFailure DELETEs every DumpTables row for target in its own
// best-effort transaction so a subsequent restore is not blocked by stale
// rows. Cleanup failures are logged; the caller still returns the original
// restore error so operators see the true cause.
func cleanupAfterFailure(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, target uuid.UUID) {
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

	// 3) Mutations only after both gates pass. Two-phase finalization: per-
	//    table transactions are unchanged; the tenant_baselines UPSERT is
	//    deferred until every table loop iteration AND every ANALYZE
	//    succeeds. A mid-restore failure triggers cleanupAfterFailure to
	//    DELETE every DumpTables row for `target` so subsequent reads see
	//    "never restored" rather than half-restored. See task-076 F5.
	if err := runRestoreTables(ctx, r.DB, tr, target); err != nil {
		r.L.WithError(err).Warnf("restore: table loop failed for target=%s region=%s ver=%d.%d; cleaning partial state", target, region, major, minor)
		cleanupAfterFailure(ctx, r.L, r.DB, target)
		return err
	}

	for _, t := range DumpTables {
		if err := r.DB.WithContext(ctx).Exec("ANALYZE " + t).Error; err != nil {
			r.L.WithError(err).Warnf("restore: ANALYZE %s failed; cleaning partial state", t)
			cleanupAfterFailure(ctx, r.L, r.DB, target)
			return err
		}
	}

	if err := r.DB.WithContext(ctx).Exec(`
        INSERT INTO tenant_baselines (tenant_id, region, major_version, minor_version, baseline_sha256, restored_at)
        VALUES (?, ?, ?, ?, ?, now())
        ON CONFLICT (tenant_id) DO UPDATE SET region=EXCLUDED.region, major_version=EXCLUDED.major_version,
            minor_version=EXCLUDED.minor_version, baseline_sha256=EXCLUDED.baseline_sha256, restored_at=now()
    `, target.String(), region, major, minor, expectedSum).Error; err != nil {
		return err
	}
	r.L.Infof("restore: finalized target=%s region=%s ver=%d.%d sha=%s", target, region, major, minor, expectedSum)
	return nil
}

func tenantColIndex(table string) int {
	switch table {
	case "documents":
		// (id, tenant_id, type, document_id, content, updated_at)
		return 1
	case "monster_search_index":
		return 0
	case "npc_search_index":
		return 0
	case "reactor_search_index":
		return 0
	case "map_search_index":
		return 0
	case "item_string_search_index":
		return 0
	}
	return 0
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

// copyInBinary pipes rw.Stream(in,…) into `COPY <table> FROM STDIN (FORMAT binary)`
// through the underlying pgx connection borrowed from the gorm transaction.
func copyInBinary(ctx context.Context, tx *gorm.DB, table string, in io.Reader, rw Rewriter) error {
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
		go func() {
			defer pw.Close()
			errc <- rw.Stream(in, pw)
		}()
		sql := fmt.Sprintf(`COPY %s FROM STDIN (FORMAT binary)`, table)
		if _, err := pgxConn.Conn().PgConn().CopyFrom(ctx, pr, sql); err != nil {
			// Drain the rewriter goroutine so it doesn't deadlock writing to pw.
			_ = pr.CloseWithError(err)
			<-errc
			return err
		}
		return <-errc
	})
}
