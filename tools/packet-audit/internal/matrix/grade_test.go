package matrix

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// helper: a one-version Inputs scaffold the cases mutate.
func baseInputs() Inputs {
	return Inputs{
		Registry: opregistry.Registry{Versions: map[string]*opregistry.VersionFile{}},
		Reports:  map[string]map[string]LoadedReport{}, // version -> writer -> report
		Routed:   map[string]map[RouteKey]bool{},       // version -> (opcode,dir) routed
		Evidence: map[EvKey]EvidenceStatus{},           // (packet,version) -> status
		Tier1:    map[string]bool{},                    // packet id -> tier1
		Markers:  map[EvKey]MarkerStatus{},             // (packet,version) -> marker
	}
}

func TestGradeNA(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t /* no entries */)
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002}, "gms_v83", false, nil)
	if c.State != StateNA {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeConflictTemplateRoutesAbsentOp(t *testing.T) {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // op absent
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x002, opregistry.DirClientbound}: true}
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002}, "gms_v83", false, nil)
	if c.State != StateConflict {
		t.Errorf("state = %v", c.State.Name())
	}
}

func TestGradeConflictAtlasClaimsAbsentOp(t *testing.T) {
	// Absent op, resolved report, and report's fname is NOT in the present-op set
	// (presentFnames=nil means no present ops) → must trigger conflict.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // absent
	in.Reports["gms_v83"] = map[string]LoadedReport{"AccountInfo": {
		WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult",
		Address:   "0xa3f2e8", // resolved address — must trigger conflict
		AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go", Verdict: diff.VerdictMatch,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CLogin::OnAccountInfoResult": "AccountInfo"}}
	// presentFnames does not contain this fname → conflict fires.
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83", false, map[string]bool{})
	if c.State != StateConflict {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeAbsentUnresolvedReportIsNA(t *testing.T) {
	// Applicability=Absent, not routed, hasReport=true but Address="ABSENT":
	// the report was not located in this version's IDB, so it does not constitute
	// Atlas claiming ownership — must grade StateNA, not StateConflict.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // absent
	in.Reports["gms_v83"] = map[string]LoadedReport{"GuildBBSListThreads": {
		WriterName: "GuildBBSListThreads", IDAName: "CUIGuildBBS::SendLoadListRequest",
		Address:   "ABSENT", // unresolved — function not found in IDB
		AtlasFile: "libs/atlas-packet/guild/serverbound/bbs_list_threads.go", Verdict: diff.VerdictBlocker,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CUIGuildBBS::SendLoadListRequest": "GuildBBSListThreads"}}
	c := gradeOpCell(in, opEntryRef{Op: "GUILD_BBS_LIST_THREADS", Dir: opregistry.DirServerbound, Opcode: 0x0E5,
		FName: "CUIGuildBBS::SendLoadListRequest"}, "gms_v83", false, nil)
	if c.State != StateNA {
		t.Errorf("absent + unresolved report must be NA; got %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeAbsentReportClaimedByPresentOpIsNA(t *testing.T) {
	// Absent op, resolved report, but reportFnameClaimedByPresentOp=true
	// (a present op in this version shares the same fname).
	// Must grade StateNA, not StateConflict — the report belongs to the present op.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t) // this op is absent
	in.Reports["gms_v83"] = map[string]LoadedReport{"Emotion": {
		WriterName: "Emotion", IDAName: "CUser::OnEmotion",
		Address:   "0xb1c2d3", // resolved address
		AtlasFile: "libs/atlas-packet/user/clientbound/emotion.go", Verdict: diff.VerdictMatch,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CUser::OnEmotion": "Emotion"}}
	// presentFnames contains "CUser::OnEmotion" — a present op in this version owns it.
	presentFnames := map[string]bool{"CUser::OnEmotion": true}
	c := gradeOpCell(in, opEntryRef{Op: "IDA_0X0E8", Dir: opregistry.DirClientbound, Opcode: 0x0E8,
		FName: "CUser::OnEmotion"}, "gms_v83", false, presentFnames)
	if c.State != StateNA {
		t.Errorf("absent + resolved report claimed by present op must be NA; got %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeConflictCrossVersionTemplateGap(t *testing.T) {
	// Registry: present in gms_v83. This version's template does NOT route it,
	// but another version (gms_v87) routes the SAME op by ITS own opcode.
	// A local audit report exists, so this IS a real template-wiring gap
	// (Atlas implements the op but the template doesn't wire the opcode).
	// Build pre-computes routedElsewhere=true for this version and passes it
	// directly to gradeOpCell (task-085 per-packet cross-version routing rule).
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[RouteKey]bool{} // not routed here
	// Provide a local report so the coverage-gap conflict fires (not mere absence).
	in.Reports["gms_v83"] = map[string]LoadedReport{"AccountInfo": {
		WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult",
		AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go", Verdict: diff.VerdictMatch,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v83": {"CLogin::OnAccountInfoResult": "AccountInfo"}}
	// routedElsewhere=true signals that another version routes this op (by its own opcode).
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83", true /* routedElsewhere */, nil)
	if c.State != StateConflict {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeCoverageGapNoReportIsIncomplete(t *testing.T) {
	// Present + routedElsewhere=true but NO local audit report → Incomplete (❌),
	// not Conflict. Without a local report Atlas hasn't implemented this op in
	// this version, so the "template coverage gap" conflict must NOT fire.
	// This is the regression guard for the 398-cell class that was previously
	// mis-graded as 🟥.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[RouteKey]bool{} // not routed here
	// No Reports / FNameToWriter set — no local report for this version.
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83", true /* routedElsewhere */, nil)
	if c.State != StateIncomplete {
		t.Errorf("coverage-gap with no report must be Incomplete (❌), not %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeUnroutedEverywhereIsIncompleteNotConflict(t *testing.T) {
	// Core regression guard for the 914-false-conflict fix (task-085):
	// an op present + unrouted in EVERY version (routedElsewhere=false) must
	// grade Incomplete, NOT Conflict.  A raw-opcode coincidence with some other
	// routed op in another version's template must NOT trigger a conflict here.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	// routedElsewhere=false: no other version routes this specific op.
	c := gradeOpCell(in, opEntryRef{Op: "ACCOUNT_INFO", Dir: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult"}, "gms_v83", false /* routedElsewhere */, nil)
	if c.State != StateIncomplete {
		t.Errorf("unrouted-everywhere must be Incomplete, not Conflict; state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradePartialToolPassNoTest(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StatePartial {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeVerifiedTier0(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Markers[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateVerified {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeTier1ToolPassCapsAtPartial(t *testing.T) {
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Tier1["login/clientbound/AccountInfo"] = true
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StatePartial {
		t.Errorf("tier1 tool-pass must cap at partial; state = %v", c.State.Name())
	}
}

func TestGradeTier1FixturePromotes(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, true) // diff verdict advisory on tier1
	in.Tier1["login/clientbound/AccountInfo"] = true
	in.Markers[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Evidence[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateVerified {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeEvidencePinnedDeferralIsPartial(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, false)
	in.Evidence[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StatePartial {
		t.Errorf("state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeStaleEvidenceDegrades(t *testing.T) {
	in := presentWithReport(t, diff.VerdictDeferred, false)
	in.Evidence[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: false}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateIncomplete {
		t.Errorf("stale evidence must degrade; state = %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeBlockerVerdictIncomplete(t *testing.T) {
	in := presentWithReport(t, diff.VerdictBlocker, false)
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateIncomplete {
		t.Errorf("state = %v", c.State.Name())
	}
}

func TestGradeUnknownVersionFile(t *testing.T) {
	in := baseInputs() // no registry file at all for gms_v84
	c := gradeOpCell(in, refACCOUNT(), "gms_v84", false, nil)
	if c.State != StateIncomplete || c.Note == "" {
		t.Errorf("unknown applicability must be incomplete+note; got %v %q", c.State.Name(), c.Note)
	}
}

func TestDemuxFamilyWorstOfNoConflict(t *testing.T) {
	// Two writers sharing the same base FName (a demux family, e.g. CUser::OnEffect
	// with #case suffixes) must grade worst-of — NOT conflict. Both must be marked
	// used so neither leaks into the sub-struct section.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x002, opregistry.DirClientbound}: true}
	// Two writers sharing the same base FName via different #case suffixes.
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"AccountInfoV1": {WriterName: "AccountInfoV1", IDAName: "CLogin::OnAccountInfoResult#A",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v1.go", Verdict: diff.VerdictMatch},
		"AccountInfoV2": {WriterName: "AccountInfoV2", IDAName: "CLogin::OnAccountInfoResult#B",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v2.go", Verdict: diff.VerdictMatch},
	}
	m := Build(in, []string{"gms_v83"})
	var cell Cell
	for _, r := range m.Rows {
		if r.Op == "ACCOUNT_INFO" {
			cell = r.Cells["gms_v83"]
			break
		}
	}
	// Worst-of two VerdictMatch-without-marker = Partial, not Conflict.
	if cell.State == StateConflict {
		t.Errorf("demux family must grade worst-of, not conflict; got %v (%s)", cell.State.Name(), cell.Note)
	}
	if cell.State != StatePartial {
		t.Errorf("demux family worst-of two Partial must be Partial; got %v (%s)", cell.State.Name(), cell.Note)
	}
	// Neither writer must leak into sub-struct rows.
	for _, r := range m.Rows {
		if r.Kind == RowSubStruct {
			t.Errorf("demux-family writers must not produce a sub-struct row; got packet=%q", r.Packet)
		}
	}
}

func TestDemuxFamilyIdenticalFullIDANameWorstOf(t *testing.T) {
	// Two writers with the IDENTICAL full IDAName (no #case suffix, same base name)
	// sharing the same op — now grades worst-of, NOT conflict.
	// The demux-family rule applies regardless of whether the names are suffixed.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002,
		FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x002, opregistry.DirClientbound}: true}
	// Both writers carry the exact same IDAName — demux-family worst-of applies.
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"AccountInfoV1": {WriterName: "AccountInfoV1", IDAName: "CLogin::OnAccountInfoResult",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v1.go", Verdict: diff.VerdictMatch},
		"AccountInfoV2": {WriterName: "AccountInfoV2", IDAName: "CLogin::OnAccountInfoResult",
			AtlasFile: "libs/atlas-packet/login/clientbound/account_info_v2.go", Verdict: diff.VerdictBlocker},
	}
	m := Build(in, []string{"gms_v83"})
	var cell Cell
	for _, r := range m.Rows {
		if r.Op == "ACCOUNT_INFO" {
			cell = r.Cells["gms_v83"]
			break
		}
	}
	// Must NOT be conflict; worst-of VerdictMatch vs VerdictBlocker = Incomplete.
	if cell.State == StateConflict {
		t.Errorf("shared full IDAName must grade worst-of, not conflict; got %v (%s)", cell.State.Name(), cell.Note)
	}
	if cell.State != StateIncomplete {
		t.Errorf("worst-of (match vs blocker) must be Incomplete; got %v (%s)", cell.State.Name(), cell.Note)
	}
	// Neither writer must leak into sub-struct rows.
	for _, r := range m.Rows {
		if r.Kind == RowSubStruct {
			t.Errorf("both writers must be marked used (no sub-struct leak); got packet=%q", r.Packet)
		}
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
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x002, opregistry.DirClientbound}: true}
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

// TestBuildPerPacketRoutingConflict verifies the per-packet cross-version routing
// rule at the Build level.  Two versions; the op is present in both.
//
//   - version A (gms_v83): opcode 0x010, ROUTED in A's template.
//   - version B (gms_v87): opcode 0x020, NOT routed in B's template.
//
// B's opcode (0x020) must NOT collide with any routed op in A's template
// (A only routes 0x010), so there is no global opcode coincidence.  Under the
// OLD rule, B's cell was Conflict due to raw-opcode union.  Under the new rule:
//
//   - A cell: routed=true → normal (no report → Incomplete).
//   - B cell: routedElsewhere=true (A routes the op) → Conflict ("template coverage gap").
func TestBuildPerPacketRoutingConflictAndFalsePositive(t *testing.T) {
	// -- Scenario 1: op present in two versions, routed in A by A's opcode
	// but NOT in B; B's opcode differs and has no raw coincidence in A.
	// B has a local report (Atlas implements it), so the coverage-gap conflict fires.
	// Expected: B cell = Conflict (coverage gap), A cell = Incomplete (no report).
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "MAP_TRANSFER_RESULT", Direction: opregistry.DirClientbound, Opcode: 0x010,
		FName: "CField::OnTransferFieldResult", Provenance: "csv-import"})
	in.Registry.Versions["gms_v87"] = vfWith(t, opregistry.Entry{
		Op: "MAP_TRANSFER_RESULT", Direction: opregistry.DirClientbound, Opcode: 0x020,
		FName: "CField::OnTransferFieldResult", Provenance: "csv-import"})
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x010, opregistry.DirClientbound}: true} // A routes its opcode
	in.Routed["gms_v87"] = map[RouteKey]bool{}                                         // B does NOT route 0x020
	// B (gms_v87) has a local report — Atlas implements this op — so the
	// coverage-gap conflict must fire (template-wiring gap, not mere absence).
	in.Reports["gms_v87"] = map[string]LoadedReport{"TransferFieldResult": {
		WriterName: "TransferFieldResult", IDAName: "CField::OnTransferFieldResult",
		AtlasFile: "libs/atlas-packet/field/clientbound/transfer_field_result.go", Verdict: diff.VerdictMatch,
	}}
	in.FNameToWriter = map[string]map[string]string{"gms_v87": {"CField::OnTransferFieldResult": "TransferFieldResult"}}

	m := Build(in, []string{"gms_v83", "gms_v87"})
	var cellA, cellB Cell
	for _, r := range m.Rows {
		if r.Op == "MAP_TRANSFER_RESULT" {
			cellA = r.Cells["gms_v83"]
			cellB = r.Cells["gms_v87"]
		}
	}
	// A: routed, no report → Incomplete (normal path).
	if cellA.State != StateIncomplete {
		t.Errorf("A cell: routed+no-report must be Incomplete; got %v (%s)", cellA.State.Name(), cellA.Note)
	}
	// B: present, not routed, routed elsewhere (in A), has report → Conflict (coverage gap).
	if cellB.State != StateConflict {
		t.Errorf("B cell: present+unrouted+routedElsewhere+hasReport must be Conflict; got %v (%s)", cellB.State.Name(), cellB.Note)
	}

	// -- Scenario 2 (MAP_TRANSFER_RESULT false-positive guard): op present in
	// two versions, routed in NEITHER; B's opcode coincides with a DIFFERENT
	// routed op's raw opcode in A's template.  Under the old rule this fired a
	// conflict; under the new rule both cells must be Incomplete.
	in2 := baseInputs()
	in2.Registry.Versions["gms_v83"] = vfWith(t,
		opregistry.Entry{
			Op: "MAP_TRANSFER_RESULT", Direction: opregistry.DirClientbound, Opcode: 0x042,
			FName: "CField::OnTransferFieldResult", Provenance: "csv-import"},
		opregistry.Entry{
			Op: "OTHER_OP", Direction: opregistry.DirClientbound, Opcode: 0x041,
			FName: "CField::OnOtherOp", Provenance: "csv-import"})
	in2.Registry.Versions["gms_v95"] = vfWith(t, opregistry.Entry{
		Op: "MAP_TRANSFER_RESULT", Direction: opregistry.DirClientbound, Opcode: 0x041,
		FName: "CField::OnTransferFieldResult", Provenance: "csv-import"})
	// A routes OTHER_OP (0x041) but NOT MAP_TRANSFER_RESULT (0x042).
	// v95's MAP_TRANSFER_RESULT opcode 0x041 coincides with A's OTHER_OP opcode.
	in2.Routed["gms_v83"] = map[RouteKey]bool{{0x041, opregistry.DirClientbound}: true} // OTHER_OP routed, NOT MAP_TRANSFER_RESULT
	in2.Routed["gms_v95"] = map[RouteKey]bool{}                                         // MAP_TRANSFER_RESULT not routed in v95 either

	m2 := Build(in2, []string{"gms_v83", "gms_v95"})
	for _, r := range m2.Rows {
		if r.Op == "MAP_TRANSFER_RESULT" {
			c83 := r.Cells["gms_v83"]
			c95 := r.Cells["gms_v95"]
			if c83.State == StateConflict {
				t.Errorf("false-positive guard: v83 MAP_TRANSFER_RESULT unrouted-everywhere must NOT be Conflict; got %v (%s)", c83.State.Name(), c83.Note)
			}
			if c95.State == StateConflict {
				t.Errorf("false-positive guard: v95 MAP_TRANSFER_RESULT unrouted-everywhere must NOT be Conflict; got %v (%s)", c95.State.Name(), c95.Note)
			}
		}
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
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x002, opregistry.DirClientbound}: true}
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
