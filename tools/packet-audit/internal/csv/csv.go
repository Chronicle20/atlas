package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Direction indicates whether packets flow from server→client or client→server.
type Direction int

const (
	DirClientbound Direction = iota
	DirServerbound
)

// Row represents one packet entry from the CSV, keyed by FName.
type Row struct {
	FName     string
	Direction Direction
	opcodes   map[string]int // "<region>:<major>" -> opcode value
}

// Opcode returns the opcode for a given region (e.g. "GMS") and major version number.
// Returns 0 if the mapping is absent.
func (r Row) Opcode(region string, major uint16) int {
	return r.opcodes[fmt.Sprintf("%s:%d", region, major)]
}

// Map is the loaded table, indexed by FName.
type Map struct {
	rows map[string]Row
}

// ByFName looks up a row by its FName (function name) value.
func (m Map) ByFName(name string) (Row, bool) {
	r, ok := m.rows[name]
	return r, ok
}

// All returns all rows in arbitrary order.
func (m Map) All() []Row {
	out := make([]Row, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, r)
	}
	return out
}

// Load parses the CSV at path and returns a Map keyed by FName.
// The file must have the real Atlas CSV shape:
//
//	Op, FName, Index, <region v<N>>, <notes>, <region v<N>>, <notes>, ...
//
// Column 1 (FName) is used as the row key. Unnamed/empty header columns are
// treated as notes and skipped. Version headers like "GMS v95" become the key
// "GMS:95"; bare "v95" is treated as "GMS:95".
func Load(path string, dir Direction) (Map, error) {
	f, err := os.Open(path)
	if err != nil {
		return Map{}, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // rows may have varying numbers of columns
	records, err := r.ReadAll()
	if err != nil {
		return Map{}, err
	}
	if len(records) < 2 {
		return Map{}, errors.New("csv: missing header or data rows")
	}

	header := records[0]

	// Locate the FName column.
	fnameCol := -1
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), "FName") {
			fnameCol = i
			break
		}
	}
	if fnameCol < 0 {
		return Map{}, errors.New("csv: FName column not found in header")
	}

	// Build a parallel slice: versions[i] = "GMS:95" (or "") for each column index.
	versions := parseVersionHeader(header)

	out := Map{rows: make(map[string]Row)}
	for _, rec := range records[1:] {
		if len(rec) <= fnameCol {
			continue
		}
		name := strings.TrimSpace(rec[fnameCol])
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		row := Row{FName: name, Direction: dir, opcodes: make(map[string]int)}
		for i, key := range versions {
			if key == "" || i >= len(rec) {
				continue
			}
			v := strings.TrimSpace(rec[i])
			if v == "" {
				continue
			}
			n, err := parseOpcode(v)
			if err != nil {
				// Skip cells that are not parseable as opcodes (e.g. placeholder text).
				continue
			}
			row.opcodes[key] = n
		}
		out.rows[name] = row
	}
	return out, nil
}

// parseVersionHeader returns a slice parallel to header.
// Element i holds "GMS:95" when header[i] is a version label like "GMS v95",
// "" for non-version columns (Op, FName, Index, or unnamed notes columns).
func parseVersionHeader(header []string) []string {
	out := make([]string, len(header))
	for i, col := range header {
		col = strings.TrimSpace(col)
		if region, major, ok := splitRegionMajor(col); ok {
			out[i] = fmt.Sprintf("%s:%d", region, major)
		}
	}
	return out
}

// splitRegionMajor parses "GMS v95" → ("GMS", 95, true) or "v95" → ("GMS", 95, true).
// Returns ("", 0, false) for columns that are not version labels.
func splitRegionMajor(col string) (string, uint16, bool) {
	parts := strings.Fields(col)
	switch len(parts) {
	case 1:
		// Bare "v95" form — assume GMS.
		if !strings.HasPrefix(parts[0], "v") && !strings.HasPrefix(parts[0], "V") {
			return "", 0, false
		}
		n, err := strconv.ParseUint(parts[0][1:], 10, 16)
		if err != nil {
			return "", 0, false
		}
		return "GMS", uint16(n), true
	case 2:
		// "<REGION> v<N>" form, e.g. "GMS v95" or "JMS v185".
		region := parts[0]
		vpart := parts[1]
		if !strings.HasPrefix(vpart, "v") && !strings.HasPrefix(vpart, "V") {
			return "", 0, false
		}
		n, err := strconv.ParseUint(vpart[1:], 10, 16)
		if err != nil {
			return "", 0, false
		}
		return region, uint16(n), true
	}
	return "", 0, false
}

// parseOpcode converts a hex string ("0x00B", "0x00") or decimal string ("11")
// into an integer.
func parseOpcode(s string) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 32)
		return int(n), err
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}
