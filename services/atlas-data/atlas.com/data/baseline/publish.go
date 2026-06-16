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
	"strings"
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

	// Resolve each table's column list up front so it can be recorded in the
	// header (written first) and reused as the COPY-out projection. The list is
	// ordered deterministically (by column name) so the dump — and therefore
	// its sha256 — is reproducible regardless of the table's physical order.
	columns := make(map[string][]string, len(DumpTables))
	for _, table := range DumpTables {
		cols, err := tableColumns(ctx, p.DB, table)
		if err != nil {
			return "", fmt.Errorf("publish: columns %s: %w", table, err)
		}
		columns[table] = cols
	}

	hdr := Header{
		SchemaVersion: SchemaVersion,
		Region:        region,
		MajorVersion:  major,
		MinorVersion:  minor,
		Tables:        DumpTables,
		Columns:       columns,
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
		if err := dumpTable(ctx, p.DB, table, columns[table], region, major, minor, tw); err != nil {
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

// tableColumns returns the table's column names in a deterministic order (by
// name). Recorded in the dump header and used as the COPY projection so restore
// can map stream fields to columns by name rather than physical position.
func tableColumns(ctx context.Context, db *gorm.DB, table string) ([]string, error) {
	var cols []string
	err := db.WithContext(ctx).Raw(
		`SELECT column_name FROM information_schema.columns
		 WHERE table_schema = current_schema() AND table_name = ?
		 ORDER BY column_name`, table).Scan(&cols).Error
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns found for table %s", table)
	}
	return cols, nil
}

func dumpTable(ctx context.Context, db *gorm.DB, table string, cols []string, region string, major, minor int, tw *tar.Writer) error {
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
		return runCopyOut(ctx, driverConn, table, cols, region, major, minor, tw)
	})
}

// runCopyOut writes `COPY (SELECT * FROM <table> WHERE tenant_id = <canonical> ORDER BY id) TO STDOUT (FORMAT binary)`
// into a tar entry <table>.binary.
//
// The full canonical subset of one table is buffered in memory (bounded by the
// PRD-mandated ~150 MB cap on canonical data) so the tar entry can be written
// with a known Size header.
func runCopyOut(ctx context.Context, driverConn any, table string, cols []string, region string, major, minor int, tw *tar.Writer) error {
	pgxConn, ok := driverConn.(*stdlib.Conn)
	if !ok {
		return fmt.Errorf("expected *stdlib.Conn, got %T", driverConn)
	}
	var buf bytes.Buffer
	if _, err := pgxConn.Conn().PgConn().CopyTo(ctx, &buf, copyOutSQL(table, cols, region, major, minor)); err != nil {
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

// copyOutSQL builds the binary COPY statement that dumps the version-scoped
// canonical-tenant subset of a table with an EXPLICIT, ordered column list
// (not `SELECT *`), ordered deterministically by a column the table actually
// has. The explicit projection pins the dump's field order to the header's
// recorded column list so restore can replay it name-keyed, immune to the
// target's physical column order. The documents table carries a surrogate
// `id`; the *_search_index tables are keyed by (tenant_id, <entity>_id) with
// no `id`, so ordering them by `id` fails with SQLSTATE 42703 — the
// publish-time empty-500 seen on atlas-main.
func copyOutSQL(table string, cols []string, region string, major, minor int) string {
	tenantId := canonical.TenantId(region, uint16(major), uint16(minor)).String()
	return fmt.Sprintf(`COPY (SELECT %s FROM %s WHERE tenant_id = '%s' ORDER BY %s) TO STDOUT (FORMAT binary)`,
		quoteCols(cols), table, tenantId, orderColumn(table))
}

// quoteCols renders a column list as a comma-separated, double-quoted
// projection (`"a","b"`). Column names come from information_schema, but
// quoting keeps the SQL safe against reserved words.
func quoteCols(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
	}
	return strings.Join(quoted, ",")
}

// orderColumn returns the column used to order a table's COPY dump. Mirrors the
// DumpTables set; the default orders by tenant_id (present on every dumped
// table) so an unmapped future table degrades to a stable-but-coarse order
// rather than crashing the dump.
func orderColumn(table string) string {
	switch table {
	case "documents":
		return "id"
	case "monster_search_index":
		return "monster_id"
	case "npc_search_index":
		return "npc_id"
	case "reactor_search_index":
		return "reactor_id"
	case "map_search_index":
		return "map_id"
	case "item_string_search_index":
		return "item_id"
	}
	return "tenant_id"
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
