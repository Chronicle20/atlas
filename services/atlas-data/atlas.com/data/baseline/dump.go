package baseline

import (
	"encoding/json"
	"fmt"
	"time"
)

// SchemaVersion bumps in lockstep with the schema-version fingerprint check.
//
// v2: dumps record an explicit per-table column list in the header and use
// name-keyed COPY (`COPY t (cols) FROM`) on restore instead of positional
// `SELECT *`. This makes restore robust to physical column-order drift between
// the publish source (atlas-main, columns appended via ALTER over time) and a
// freshly-AutoMigrated restore target (columns in struct order). v1 dumps are
// positional and are rejected by the restore schema gate.
const SchemaVersion = "v2"

// DumpTables is the ordered list of tables included in a baseline dump.
var DumpTables = []string{
	"documents",
	"monster_search_index",
	"npc_search_index",
	"reactor_search_index",
	"map_search_index",
	"item_string_search_index",
}

// Header is the deterministic JSON entry written as the first tar entry.
//
// Columns records, per table, the exact ordered column list the dump's binary
// COPY stream was produced with. Restore replays it as `COPY <table> (cols)
// FROM STDIN` so Postgres maps stream fields to columns by NAME, immune to the
// target table's physical column order.
type Header struct {
	SchemaVersion string              `json:"schemaVersion"`
	Region        string              `json:"region"`
	MajorVersion  int                 `json:"majorVersion"`
	MinorVersion  int                 `json:"minorVersion"`
	Tables        []string            `json:"tables"`
	Columns       map[string][]string `json:"columns"`
	PublishedAt   time.Time           `json:"publishedAt"`
}

// MarshalHeader encodes the header as canonical JSON.
func MarshalHeader(h Header) ([]byte, error) {
	return json.Marshal(h)
}

// DumpKey returns the MinIO object key for the baseline tar.
func DumpKey(region string, major, minor int) string {
	return fmt.Sprintf("baseline/regions/%s/versions/%d.%d/documents.dump", region, major, minor)
}

// ShaKey returns the MinIO object key for the sha256 sidecar.
func ShaKey(region string, major, minor int) string {
	return fmt.Sprintf("baseline/regions/%s/versions/%d.%d/documents.dump.sha256", region, major, minor)
}
