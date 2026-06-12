package matrix

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// helper: a one-version Inputs scaffold the cases mutate.
func baseInputs() Inputs {
	return Inputs{
		Registry:       opregistry.Registry{Versions: map[string]*opregistry.VersionFile{}},
		Reports:        map[string]map[string]LoadedReport{}, // version -> writer -> report
		Routed:         map[string]map[routeKey]bool{},       // version -> (opcode,dir) routed
		RoutedAnywhere: map[routeKey]bool{},                  // (opcode,dir) routed in ANY version
		Evidence:       map[evKey]EvidenceStatus{},           // (packet,version) -> status
		Tier1:          map[string]bool{},                    // packet id -> tier1
		Markers:        map[evKey]MarkerStatus{},             // (packet,version) -> marker
	}
}

func TestGradeNA(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t /* no entries */)
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002}, "gms_v83")
	if c.State != StateNA {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeConflictTemplateRoutesAbsentOp(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // op absent
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002}, "gms_v83")
	if c.State != StateConflict {
		t.Errorf("state = %v", c.State.Name())
	}
}

func TestGradeConflictAtlasClaimsAbsentOp(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // absent
	in.Reports["gms_v83"] = map[string]LoadedReport{"AccountInfo": {
		WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult",
		AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go", Verdict: diff.VerdictMatch,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CLogin::OnAccountInfoResult": "AccountInfo"}}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83")
	if c.State != StateConflict {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeConflictCrossVersionTemplateGap(t *testing.T) {
	// Registry: present. This version's template does NOT route it, but some
	// other version's template does -> the task-067/068 gap class (context.md
	// decision D3 refines design §5).
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{} // not routed here
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83")
	if c.State != StateConflict {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeUnroutedEverywhereIsIncompleteNotConflict(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83")
	if c.State != StateIncomplete {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradePartialToolPassNoTest(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StatePartial {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeVerifiedTier0(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Markers[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateVerified {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeTier1ToolPassCapsAtPartial(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Tier1["login/clientbound/AccountInfo"] = true
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StatePartial {
		t.Errorf("tier1 tool-pass must cap at partial; state = %v", c.State.Name())
	}
}

func TestGradeTier1FixturePromotes(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, true) // diff verdict advisory on tier1
	in.Tier1["login/clientbound/AccountInfo"] = true
	in.Markers[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Evidence[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateVerified {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeEvidencePinnedDeferralIsPartial(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, false)
	in.Evidence[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StatePartial {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeStaleEvidenceDegrades(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, false)
	in.Evidence[evKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: false}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateIncomplete {
		t.Errorf("stale evidence must degrade; state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeBlockerVerdictIncomplete(t *testing.T) {
	in := presentWithReport(t, diff.VerdictBlocker, false)
	c := gradeOpCell(in, refACCOUNT(), "gms_v83")
	if c.State != StateIncomplete {
		t.Errorf("state = %v", c.State.Name())
	}
}

func TestGradeUnknownVersionFile(t *testing.T) {
	in := baseInputs() // no registry file at all for gms_v84
	c := gradeOpCell(in, refACCOUNT(), "gms_v84")
	if c.State != StateIncomplete || c.Note == "" {
		t.Errorf("unknown applicability must be incomplete+note; got %v %q", c.State.Name(), c.Note)
	}
}

func TestGradeConflictDuplicateClaim(t *testing.T) {
	// Two different writers with the IDENTICAL full IDAName (no #case suffix)
	// for the same op -> StateConflict.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	// Two writers both claim the exact same IDAName (no #case suffix).
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"AccountInfoV1": {WriterName: "AccountInfoV1", IDAName: "CLogin::OnAccountInfoResult",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v1.go", Verdict: diff.VerdictMatch},
		"AccountInfoV2": {WriterName: "AccountInfoV2", IDAName: "CLogin::OnAccountInfoResult",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v2.go", Verdict: diff.VerdictMatch},
	}
	// Both writers map to the same baseFName = "CLogin::OnAccountInfoResult".
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {
		"CLogin::OnAccountInfoResult": "AccountInfoV1",
	}}
	m := Build(in, []string{"gms_v83"})
	var cell Cell
	for _, r := range m.Rows {
		if r.Op == "ACCOUNT_INFO" {
			cell = r.Cells["gms_v83"]
			break
		}
	}
	if cell.State != StateConflict {
		t.Errorf("duplicate-claim must be conflict; got %v (%s)", cell.State.Name(), cell.Note)
	}
	if !strings.Contains(cell.Note, "two Atlas structs claim") {
		t.Errorf("conflict note must mention 'two Atlas structs claim'; got %q", cell.Note)
	}
}

func TestDuplicateClaimNoSubStructLeak(t *testing.T) {
	// When two writers carry the same IDAName (duplicate claim → Conflict on the
	// op row), both must be marked used so neither appears as a sub-struct row.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"AccountInfoV1": {WriterName: "AccountInfoV1", IDAName: "CLogin::OnAccountInfoResult",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v1.go", Verdict: diff.VerdictMatch},
		"AccountInfoV2": {WriterName: "AccountInfoV2", IDAName: "CLogin::OnAccountInfoResult",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v2.go", Verdict: diff.VerdictMatch},
	}
	m := Build(in, []string{"gms_v83"})
	for _, r := range m.Rows {
		if r.Kind == RowSubStruct {
			t.Errorf("duplicate-claim writers must not produce a sub-struct row; got packet=%q", r.Packet)
		}
	}
}

func TestDuplicateClaimWithCaseSuffix(t *testing.T) {
	// Two DIFFERENT writers carrying the same full IDAName INCLUDING a #case
	// suffix (e.g. both named "CFoo::OnBar#A") is also a duplicate claim.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	// Both writers share the same full IDAName (with #case suffix).
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"AccountInfoA": {WriterName: "AccountInfoA", IDAName: "CLogin::OnAccountInfoResult#Invite",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_a.go", Verdict: diff.VerdictMatch},
		"AccountInfoB": {WriterName: "AccountInfoB", IDAName: "CLogin::OnAccountInfoResult#Invite",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_b.go", Verdict: diff.VerdictMatch},
	}
	m := Build(in, []string{"gms_v83"})
	var cell Cell
	for _, r := range m.Rows {
		if r.Op == "ACCOUNT_INFO" {
			cell = r.Cells["gms_v83"]
		}
	}
	if cell.State != StateConflict {
		t.Errorf("duplicate #case IDAName must be conflict; got %v (%s)", cell.State.Name(), cell.Note)
	}
}

func TestWorstOfBlockerWinsOverMatch(t *testing.T) {
	// Two per-case writers of one dispatcher FName: one grades Incomplete
	// (VerdictBlocker), the other grades Partial (VerdictMatch without marker).
	// The op row must be Incomplete — Blocker must win under severity().
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CFoo::OnBar", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"FooBarA": {WriterName: "FooBarA", IDAName: "CFoo::OnBar#A",
			AtlasFile: "libs/atlas-packet/foo/clientbound/foo_bar_a.go", Verdict: diff.VerdictMatch},
		"FooBarB": {WriterName: "FooBarB", IDAName: "CFoo::OnBar#B",
			AtlasFile: "libs/atlas-packet/foo/clientbound/foo_bar_b.go", Verdict: diff.VerdictBlocker},
	}
	m := Build(in, []string{"gms_v83"})
	var cell Cell
	for _, r := range m.Rows {
		if r.Op == "ACCOUNT_INFO" {
			cell = r.Cells["gms_v83"]
		}
	}
	if cell.State != StateIncomplete {
		t.Errorf("Blocker case must win worst-of (Incomplete); got %v (%s)", cell.State.Name(), cell.Note)
	}
}

// --- helpers ---

func refACCOUNT() opEntryRef {
	return opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}
}

func presentWithReport(t *testing.T, v diff.Verdict, flatInvalid bool) Inputs {
	t.Helper()
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.RoutedAnywhere = map[routeKey]bool{{0x002, opregistry.DirClientbound}: true}
	in.Reports["gms_v83"] = map[string]LoadedReport{"AccountInfo": {
		WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult", Address: "0xa3f2e8",
		AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go",
		Verdict:   v, FlatInvalid: flatInvalid,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CLogin::OnAccountInfoResult": "AccountInfo"}}
	return in
}

// vfWith builds a VersionFile from entries via LoadVersion round-trip semantics.
func vfWith(t *testing.T, entries ...opregistry.Entry) *opregistry.VersionFile {
	t.Helper()
	return opregistry.NewVersionFile(entries)
}
