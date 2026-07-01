// Package matrix joins registry applicability, audit verdicts, evidence,
// tier membership and byte-test linkage into the coverage matrix
// (task-085 design §4, §5, §9).
package matrix

import (
	"encoding/json"
	"fmt"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// VersionKeys is the canonical baseline column order (design §3).
var VersionKeys = []string{"gms_v72", "gms_v79", "gms_v83", "gms_v84", "gms_v87", "gms_v95", "jms_v185"}

// ExportPath maps a version key to its IDA export JSON. jms_v185's export
// kept its historical gms_jms_185 name (see memory: jms audit-dir mismatch).
func ExportPath(versionKey string) string {
	if versionKey == "jms_v185" {
		return "docs/packets/ida-exports/gms_jms_185.json"
	}
	return "docs/packets/ida-exports/" + versionKey + ".json"
}

// templateFiles maps a version key to its tenant seed-template filename.
// Guarded by TestEveryVersionKeyHasTemplateFile so a new VersionKeys entry that
// forgets its template here fails `go test` instead of silently emitting a
// "no template for <key>" matrix warning + an unrouted (wrong-applicability) column.
var templateFiles = map[string]string{
	"gms_v72":  "template_gms_72_1.json",
	"gms_v79":  "template_gms_79_1.json",
	"gms_v83":  "template_gms_83_1.json",
	"gms_v84":  "template_gms_84_1.json",
	"gms_v87":  "template_gms_87_1.json",
	"gms_v95":  "template_gms_95_1.json",
	"jms_v185": "template_jms_185_1.json",
}

// TemplatePath maps a version key to the tenant seed template.
// gms_v83 -> services/atlas-configurations/seed-data/templates/template_gms_83_1.json
func TemplatePath(versionKey string) string {
	return "services/atlas-configurations/seed-data/templates/" + templateFiles[versionKey]
}

// State is the graded cell state. The declared order is the design §5
// precedence order (NA < Conflict < Verified < Partial < Incomplete) and is
// part of the status.json contract — do NOT reorder it; append new states at
// the END so existing numeric values stay stable. For worst-of comparisons use
// severity(), not the numeric value.
type State int

const (
	StateNA State = iota
	StateConflict
	StateVerified
	StatePartial
	StateIncomplete
	// StateFamily: the op is a mode-prefix DISPATCHER (one opcode, a leading
	// mode/discriminator byte switching to many sub-handlers with distinct
	// bodies). A byte-fixture proves only the leading byte + the one fixtured
	// sub-handler — NOT the remaining mode arms — so such an op is capped here
	// and can never reach ✅ on a single sub-handler. Appended last to preserve
	// the status.json numeric contract.
	StateFamily
)

// severity maps a State to its worst-of rank. Conflict is the most severe
// (must win the worst-of comparison), then Incomplete, Partial, Family,
// Verified, NA. Family ranks above Verified so a dispatcher op never presents
// as verified when any candidate is capped, but below Partial/Incomplete so a
// genuine gap on the same op still surfaces.
func severity(s State) int {
	switch s {
	case StateConflict:
		return 5
	case StateIncomplete:
		return 4
	case StatePartial:
		return 3
	case StateFamily:
		return 2
	case StateVerified:
		return 1
	default: // StateNA
		return 0
	}
}

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
	case StateFamily:
		return "🧩"
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
	case StateFamily:
		return "family"
	default:
		return "incomplete"
	}
}

func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Name())
}

func (s *State) UnmarshalJSON(b []byte) error {
	var name string
	if err := json.Unmarshal(b, &name); err != nil {
		return err
	}
	switch name {
	case "n-a":
		*s = StateNA
	case "conflict":
		*s = StateConflict
	case "verified":
		*s = StateVerified
	case "partial":
		*s = StatePartial
	case "incomplete":
		*s = StateIncomplete
	case "family":
		*s = StateFamily
	default:
		return fmt.Errorf("unknown State %q", name)
	}
	return nil
}

// Cell is one graded (op|packet, direction, version) cell.
// Opcode is the per-version registry opcode for op rows, or -1 when the op is
// absent from that version's registry or for sub-struct rows.
type Cell struct {
	State  State  `json:"state"`
	Note   string `json:"note,omitempty"` // conflict detail / degradation reason
	Opcode int    `json:"opcode,omitempty"`
}

// RowKind separates op rows (registry-joined) from sub-struct rows
// (audited shared structures with no opcode — design §10 rule 4).
type RowKind int

const (
	RowOp RowKind = iota
	RowSubStruct
)

func (k RowKind) kindName() string {
	if k == RowSubStruct {
		return "sub-struct"
	}
	return "op"
}

func (k RowKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.kindName())
}

func (k *RowKind) UnmarshalJSON(b []byte) error {
	var name string
	if err := json.Unmarshal(b, &name); err != nil {
		return err
	}
	switch name {
	case "op":
		*k = RowOp
	case "sub-struct":
		*k = RowSubStruct
	default:
		return fmt.Errorf("unknown RowKind %q", name)
	}
	return nil
}

type MatrixRow struct {
	Kind      RowKind              `json:"kind"`
	Op        string               `json:"op,omitempty"`        // RowOp only
	Packet    string               `json:"packet,omitempty"`    // "buddy/clientbound/Invite" when an Atlas struct exists
	Direction opregistry.Direction `json:"direction,omitempty"` // empty for sub-struct rows
	Tier1     bool                 `json:"tier1"`
	FNames    []string             `json:"fnames,omitempty"` // distinct base FNames across versions where op is present
	Cells     map[string]Cell      `json:"cells"`            // version key -> cell
}

// Matrix is the full joined result.
type Matrix struct {
	ToolSHA      string            `json:"toolSha"`      // git SHA of the tools/packet-audit tree
	ExportHashes map[string]string `json:"exportHashes"` // version key -> sha256 of export file
	Rows         []MatrixRow       `json:"rows"`
}
