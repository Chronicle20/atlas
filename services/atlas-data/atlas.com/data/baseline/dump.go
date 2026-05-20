package baseline

import (
	"encoding/json"
	"fmt"
	"time"
)

// SchemaVersion bumps in lockstep with the schema-version fingerprint check.
const SchemaVersion = "v1"

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
type Header struct {
	SchemaVersion string    `json:"schemaVersion"`
	Region        string    `json:"region"`
	MajorVersion  int       `json:"majorVersion"`
	MinorVersion  int       `json:"minorVersion"`
	Tables        []string  `json:"tables"`
	PublishedAt   time.Time `json:"publishedAt"`
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
