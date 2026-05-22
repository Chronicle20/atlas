package baseline

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

// Publish builds a tar of header.json + one COPY-binary entry per table and
// uploads the tar + sha256 sidecar to the canonical bucket. Returns the
// hex-encoded sha256 of the tar.
func (p Publisher) Publish(ctx context.Context, region string, major, minor int) (string, error) {
	pr, pw := io.Pipe()
	h := sha256.New()
	tw := tar.NewWriter(io.MultiWriter(pw, h))
	errc := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer tw.Close()
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
			errc <- err
			return
		}
		if err := writeTarEntry(tw, "header.json", hdrBytes); err != nil {
			errc <- err
			return
		}
		for _, table := range DumpTables {
			if err := dumpTable(ctx, p.DB, table, tw); err != nil {
				errc <- err
				return
			}
		}
		errc <- nil
	}()
	if err := p.MC.Put(ctx, p.MC.Cfg().BucketCanonical, DumpKey(region, major, minor), pr, -1, "application/x-tar"); err != nil {
		return "", err
	}
	if err := <-errc; err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if err := p.MC.Put(ctx, p.MC.Cfg().BucketCanonical, ShaKey(region, major, minor), strReader(sum), int64(len(sum)), "text/plain"); err != nil {
		return "", err
	}
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
