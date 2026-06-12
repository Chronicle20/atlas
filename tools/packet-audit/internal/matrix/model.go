// Package matrix joins registry applicability, audit verdicts, evidence,
// tier membership and byte-test linkage into the coverage matrix
// (task-085 design §4, §5, §9).
package matrix

import (
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// VersionKeys is the canonical baseline column order (design §3).
var VersionKeys = []string{"gms_v83", "gms_v84", "gms_v87", "gms_v95", "jms_v185"}

// ExportPath maps a version key to its IDA export JSON. jms_v185's export
// kept its historical gms_jms_185 name (see memory: jms audit-dir mismatch).
func ExportPath(versionKey string) string {
	if versionKey == "jms_v185" {
		return "docs/packets/ida-exports/gms_jms_185.json"
	}
	return "docs/packets/ida-exports/" + versionKey + ".json"
}

// TemplatePath maps a version key to the tenant seed template.
// gms_v83 -> services/atlas-configurations/seed-data/templates/template_gms_83_1.json
func TemplatePath(versionKey string) string {
	name := map[string]string{
		"gms_v83":  "template_gms_83_1.json",
		"gms_v84":  "template_gms_84_1.json",
		"gms_v87":  "template_gms_87_1.json",
		"gms_v95":  "template_gms_95_1.json",
		"jms_v185": "template_jms_185_1.json",
	}[versionKey]
	return "services/atlas-configurations/seed-data/templates/" + name
}

// State is the graded cell state, in design §5 precedence order.
type State int

const (
	StateNA State = iota
	StateConflict
	StateVerified
	StatePartial
	StateIncomplete
)

func (s State) Symbol() string {
	switch s {
	case StateNA:
		return "⬜"
	case StateConflict:
		return "🟥"
	case StateVerified:
		return "✅"
	case StatePartial:
		return "🟡"
	default:
		return "❌"
	}
}

func (s State) Name() string {
	switch s {
	case StateNA:
		return "n-a"
	case StateConflict:
		return "conflict"
	case StateVerified:
		return "verified"
	case StatePartial:
		return "partial"
	default:
		return "incomplete"
	}
}

// Cell is one graded (op|packet, direction, version) cell.
type Cell struct {
	State State  `json:"state"`
	Note  string `json:"note,omitempty"` // conflict detail / degradation reason
}

// RowKind separates op rows (registry-joined) from sub-struct rows
// (audited shared structures with no opcode — design §10 rule 4).
type RowKind int

const (
	RowOp RowKind = iota
	RowSubStruct
)

type MatrixRow struct {
	Kind      RowKind              `json:"kind"`
	Op        string               `json:"op,omitempty"`     // RowOp only
	Packet    string               `json:"packet,omitempty"` // "buddy/clientbound/Invite" when an Atlas struct exists
	Direction opregistry.Direction `json:"direction"`
	Tier1     bool                 `json:"tier1"`
	Cells     map[string]Cell      `json:"cells"` // version key -> cell
}

// Matrix is the full joined result.
type Matrix struct {
	ToolSHA      string            `json:"toolSha"`      // git SHA of the tools/packet-audit tree
	ExportHashes map[string]string `json:"exportHashes"` // version key -> sha256 of export file
	Rows         []MatrixRow       `json:"rows"`
}
