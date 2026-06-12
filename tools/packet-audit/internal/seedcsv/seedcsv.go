// Package seedcsv parses the hand-maintained MapleStory Ops CSVs with full
// presence semantics (per-version index+opcode pairs) for the one-time
// registry seeding (design §5.1 "CSV seeding").
//
// CSV shape: Op,FName,Index,<REGION vN>,,<REGION vN>,, ...
// A version's pair is (column before the labeled column, labeled column) =
// (index, opcode). present ⇔ index non-empty OR opcode != 0.
package seedcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Cell struct {
	Present bool
	Opcode  int
}

type Row struct {
	Op        string
	FName     string
	FNameAlts []string
	Versions  map[string]Cell // "REGION:major" -> cell
}

func Load(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return load(f, path)
}

func LoadFromString(s string) ([]Row, error) {
	return load(strings.NewReader(s), "<string>")
}

func load(src io.Reader, name string) ([]Row, error) {
	r := csv.NewReader(src)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("%s: missing header or data rows", name)
	}
	header := records[0]

	// versionCols[i] = "REGION:major" when header[i] is the OPCODE column of
	// that version; the index column is i-1.
	versionCols := make([]string, len(header))
	for i, col := range header {
		if region, major, ok := splitRegionMajor(strings.TrimSpace(col)); ok {
			versionCols[i] = fmt.Sprintf("%s:%d", region, major)
		}
	}

	var out []Row
	for rowNum, rec := range records[1:] {
		if len(rec) < 2 {
			continue
		}
		op := strings.TrimSpace(rec[0])
		if op == "" || strings.HasPrefix(op, "#") {
			continue
		}
		fnameLines := strings.Split(strings.TrimSpace(rec[1]), "\n")
		row := Row{Op: op, FName: strings.TrimSpace(fnameLines[0]), Versions: map[string]Cell{}}
		for _, alt := range fnameLines[1:] {
			if a := strings.TrimSpace(alt); a != "" {
				row.FNameAlts = append(row.FNameAlts, a)
			}
		}
		for i, vk := range versionCols {
			if vk == "" || i >= len(rec) {
				continue
			}
			opcodeCell := strings.TrimSpace(rec[i])
			indexCell := ""
			if i-1 >= 0 && i-1 < len(rec) {
				indexCell = strings.TrimSpace(rec[i-1])
			}
			if opcodeCell == "" && indexCell == "" {
				continue // ragged short row tail
			}
			opcode := 0
			if opcodeCell != "" {
				var parseErr error
				opcode, parseErr = parseOpcode(opcodeCell)
				if parseErr != nil {
					return nil, fmt.Errorf("%s row %d (%s) col %d: unparseable opcode %q",
						name, rowNum+2, op, i+1, opcodeCell)
				}
			}
			row.Versions[vk] = Cell{Present: indexCell != "" || opcode != 0, Opcode: opcode}
		}
		out = append(out, row)
	}
	return out, nil
}

// splitRegionMajor mirrors internal/csv: "GMS v95"/"JMS v185" and bare "v95"
// (assumed GMS) are version labels; everything else (Op, FName, Index, "") is not.
func splitRegionMajor(col string) (string, uint16, bool) {
	parts := strings.Fields(col)
	var region, vpart string
	switch len(parts) {
	case 1:
		region, vpart = "GMS", parts[0]
	case 2:
		region, vpart = parts[0], parts[1]
	default:
		return "", 0, false
	}
	if !strings.HasPrefix(vpart, "v") && !strings.HasPrefix(vpart, "V") {
		return "", 0, false
	}
	n, err := strconv.ParseUint(vpart[1:], 10, 16)
	if err != nil {
		return "", 0, false
	}
	return region, uint16(n), true
}

func parseOpcode(s string) (int, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 32)
		return int(n), err
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}
