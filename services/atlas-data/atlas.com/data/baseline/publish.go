package baseline

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"atlas-data/canonical"
	minio "atlas-data/storage/minio"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Publisher writes a canonical baseline dump to MinIO.
type Publisher struct {
	DB *gorm.DB
	MC *minio.Client
	L  logrus.FieldLogger
}

// Publish builds a tar of header.json + one COPY-binary entry per table to
// a temp file, computes the sha256, uploads the tar plus a sha sidecar to
// the canonical bucket, and returns the hex-encoded sha256.
//
// The earlier implementation streamed via io.Pipe directly into MC.Put;
// when any step in the writer goroutine errored, MC.Put could finish with
// the half-written body and the handler would surface an empty error.
// Buffering to a temp file makes the steps observable and lets MC.Put run
// with a known Content-Length.
func (p Publisher) Publish(ctx context.Context, region string, major, minor int) (string, error) {
	if p.DB == nil {
		return "", fmt.Errorf("publish: nil-db")
	}
	if p.MC == nil {
		return "", fmt.Errorf("publish: nil-mc")
	}
	p.L.Infof("publish: start region=%s ver=%d.%d", region, major, minor)

	tmp, err := os.CreateTemp("", "baseline-publish-*.tar")
	if err != nil {
		return "", fmt.Errorf("publish: create-tempfile: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	h := sha256.New()
	tw := tar.NewWriter(io.MultiWriter(tmp, h))

	hdr := Header{
		SchemaVersion: SchemaVersion,
		Region:        region,
		MajorVersion:  major,
		MinorVersion:  minor,
		Tables:        DumpTables,
		PublishedAt:   time.Unix(0, 0).UTC(),
	}
	hdrBytes, err := MarshalHeader(hdr)
	if err != nil {
		return "", fmt.Errorf("publish: marshal-header: %w", err)
	}
	if err := writeTarEntry(tw, "header.json", hdrBytes); err != nil {
		return "", fmt.Errorf("publish: write-header: %w", err)
	}
	for _, table := range DumpTables {
		p.L.Debugf("publish: dump-table %s", table)
		if err := dumpTable(ctx, p.DB, table, tw); err != nil {
			return "", fmt.Errorf("publish: dump-table %s: %w", table, err)
		}
	}
	if err := tw.Close(); err != nil {
		return "", fmt.Errorf("publish: close-tar: %w", err)
	}

	size, err := tmp.Seek(0, io.SeekEnd)
	if err != nil {
		return "", fmt.Errorf("publish: seek-end: %w", err)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("publish: seek-start: %w", err)
	}

	p.L.Infof("publish: upload tar size=%d", size)
	if err := p.MC.Put(ctx, p.MC.Cfg().BucketCanonical, DumpKey(region, major, minor), tmp, size, "application/x-tar"); err != nil {
		return "", fmt.Errorf("publish: put-tar: %w", err)
	}

	sum := hex.EncodeToString(h.Sum(nil))
	if err := p.MC.Put(ctx, p.MC.Cfg().BucketCanonical, ShaKey(region, major, minor), strReader(sum), int64(len(sum)), "text/plain"); err != nil {
		return "", fmt.Errorf("publish: put-sha: %w", err)
	}
	p.L.Infof("publish: ok sha=%s", sum)
	return sum, nil
}

func dumpTable(ctx context.Context, db *gorm.DB, table string, tw *tar.Writer) error {
	raw, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := raw.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		return runCopyOut(ctx, driverConn, table, tw)
	})
}

// runCopyOut writes `COPY (SELECT * FROM <table> WHERE tenant_id = <canonical> ORDER BY id) TO STDOUT (FORMAT binary)`
// into a tar entry <table>.binary.
//
// The full canonical subset of one table is buffered in memory (bounded by the
// PRD-mandated ~150 MB cap on canonical data) so the tar entry can be written
// with a known Size header.
func runCopyOut(ctx context.Context, driverConn any, table string, tw *tar.Writer) error {
	pgxConn, ok := driverConn.(*stdlib.Conn)
	if !ok {
		return fmt.Errorf("expected *stdlib.Conn, got %T", driverConn)
	}
	var buf bytes.Buffer
	sql := fmt.Sprintf(`COPY (SELECT * FROM %s WHERE tenant_id = '%s' ORDER BY id) TO STDOUT (FORMAT binary)`,
		table, canonical.TenantUUID)
	if _, err := pgxConn.Conn().PgConn().CopyTo(ctx, &buf, sql); err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: table + ".binary",
		Size: int64(buf.Len()),
		Mode: 0o644,
	}); err != nil {
		return err
	}
	_, err := tw.Write(buf.Bytes())
	return err
}

func strReader(s string) io.Reader { return bytes.NewReader([]byte(s)) }

func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Size: int64(len(data)),
		Mode: 0o644,
	}); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
