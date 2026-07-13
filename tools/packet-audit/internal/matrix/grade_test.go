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

func TestGradeFamilyCapsTier0VerifiedToFamily(t *testing.T) {
	// A would-be tier-0 ✅ (tool match + marker) is capped at 🧩 family when the
	// op's FName is a registered mode-prefix dispatcher.
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Markers[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Families = map[string]bool{"CLogin::OnAccountInfoResult": true}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateFamily {
		t.Errorf("dispatcher op must cap at family, not %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeFamilyModeEnumerationDoesNotLiftToVerified(t *testing.T) {
	// Regression for the false-pass removal: a mode-prefix dispatcher with a
	// single sub-handler fixture (marker + fresh evidence) must STAY 🧩 family.
	// Per-version mode-byte enumeration no longer lifts a dispatcher to ✅ —
	// only per-mode body coverage may (a future model), never enumeration alone.
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Families = map[string]bool{"CLogin::OnAccountInfoResult": true}
	in.Markers[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Evidence[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0xa3f2e8"}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateFamily {
		t.Errorf("dispatcher must stay 🧩 family (no enumeration lift), not %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeFamilyCapsTier1FixtureToFamily(t *testing.T) {
	// A would-be tier-1 ✅ (marker + fresh evidence) is capped at 🧩 family for a
	// dispatcher op — one sub-handler's fixture cannot verify the whole family.
	in := presentWithReport(t, diff.VerdictDeferred, true)
	in.Tier1["login/clientbound/AccountInfo"] = true
	in.Markers[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Evidence[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0xa3f2e8"}
	in.Families = map[string]bool{"CLogin::OnAccountInfoResult": true}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateFamily {
		t.Errorf("dispatcher op must cap at family, not %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeNonFamilyStillVerifies(t *testing.T) {
	// Control: with a Families set that does NOT contain this op's FName, the
	// op promotes to ✅ exactly as before (no over-capping).
	in := presentWithReport(t, diff.VerdictMatch, false)
	in.Markers[EvKey{"login/clientbound/AccountInfo", "gms_v83"}] = MarkerStatus{Found: true, Address: "0xa3f2e8"}
	in.Families = map[string]bool{"CSomethingElse::OnPacket": true}
	c := gradeOpCell(in, refACCOUNT(), "gms_v83", false, nil)
	if c.State != StateVerified {
		t.Errorf("non-dispatcher op must verify, not %v (%s)", c.State.Name(), c.Note)
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

// TestBuildRoutedNamesOpIdentityGuard exercises the op-identity guard added in
// the task-096 matrix-grader fix (build.go RoutedNames branch). The guard makes
// the cross-version routedElsewhere signal op-identity-aware: a version only
// counts as routing an op if the template's routed NAME at that opcode matches
// the op's own writer/handler (via fnameWriters). A raw-opcode coincidence —
// the opcode is occupied by a DIFFERENT op's handler in that version — must NOT
// fabricate a routedElsewhere → template-wiring Conflict for the sibling
// versions.
//
// Scenario (mirrors the fix's commit message): a serverbound WEDDING_ACTION op
// whose opcode (0x8B) in version A happens to equal A's CharacterKeyMapChange
// handler slot. A routes 0x8B — but to the KeyMapChange handler, not to
// WEDDING_ACTION. Version B has WEDDING_ACTION at a different opcode (0x90),
// implements it (has a report), but does not route it.
//
//   - Negative (guard active): A's routed NAME at 0x8B != A's WEDDING_ACTION
//     writer → A is NOT counted as routing WEDDING_ACTION → B is NOT
//     routedElsewhere → B grades Incomplete, not a false Conflict.
//   - Positive (name matches): A's routed NAME at 0x8B IS A's WEDDING_ACTION
//     writer → A counts as routing the op → B is routedElsewhere → B (which has
//     a report) grades Conflict (real template-wiring gap).
func TestBuildRoutedNamesOpIdentityGuard(t *testing.T) {
	const (
		weddingFName       = "CWvsContext::OnWeddingAction"
		weddingWriterA     = "WeddingActionA"
		weddingWriterB     = "WeddingActionB"
		keyMapChangeWriter = "CharacterKeyMapChange" // unrelated handler occupying 0x8B in A
	)
	// build constructs the two-version Inputs. routedNameAt8B is the writer/handler
	// NAME that version A's template routes opcode 0x8B to.
	build := func(routedNameAt8B string) Matrix {
		in := baseInputs()
		// Version A (gms_v83): WEDDING_ACTION serverbound at the colliding opcode 0x8B.
		in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
			Op: "WEDDING_ACTION", Direction: opregistry.DirServerbound, Opcode: 0x8B,
			FName: weddingFName, Provenance: "csv-import"})
		// Version B (gms_v87): WEDDING_ACTION serverbound at a DIFFERENT opcode 0x90.
		in.Registry.Versions["gms_v87"] = vfWith(t, opregistry.Entry{
			Op: "WEDDING_ACTION", Direction: opregistry.DirServerbound, Opcode: 0x90,
			FName: weddingFName, Provenance: "csv-import"})
		// A routes opcode 0x8B (occupancy true); the NAME it routes to varies per case.
		in.Routed["gms_v83"] = map[RouteKey]bool{{0x8B, opregistry.DirServerbound}: true}
		in.RoutedNames = map[string]map[RouteKey]string{
			"gms_v83": {{0x8B, opregistry.DirServerbound}: routedNameAt8B},
		}
		// B does NOT route its opcode 0x90.
		in.Routed["gms_v87"] = map[RouteKey]bool{}
		// Both versions implement WEDDING_ACTION (have a local report) so the guard's
		// "this version maps the op's FName to a writer" precondition holds for A and
		// the coverage-gap conflict can fire for B.
		in.Reports["gms_v83"] = map[string]LoadedReport{weddingWriterA: {
			WriterName: weddingWriterA, IDAName: weddingFName,
			AtlasFile: "libs/atlas-packet/wedding/serverbound/wedding_action.go", Verdict: diff.VerdictMatch,
		}}
		in.Reports["gms_v87"] = map[string]LoadedReport{weddingWriterB: {
			WriterName: weddingWriterB, IDAName: weddingFName,
			AtlasFile: "libs/atlas-packet/wedding/serverbound/wedding_action.go", Verdict: diff.VerdictMatch,
		}}
		in.FNameToWriter = map[string]map[string]string{
			"gms_v83": {weddingFName: weddingWriterA},
			"gms_v87": {weddingFName: weddingWriterB},
		}
		return Build(in, []string{"gms_v83", "gms_v87"})
	}

	cellsFor := func(m Matrix) (Cell, Cell) {
		var cA, cB Cell
		for _, r := range m.Rows {
			if r.Op == "WEDDING_ACTION" {
				cA = r.Cells["gms_v83"]
				cB = r.Cells["gms_v87"]
			}
		}
		return cA, cB
	}

	t.Run("coincidence-rejected", func(t *testing.T) {
		// A routes 0x8B to an UNRELATED handler (KeyMapChange) — a raw-opcode
		// coincidence. The guard must reject A as routing WEDDING_ACTION, so B is
		// NOT routedElsewhere and must NOT be a false Conflict.
		cA, cB := cellsFor(build(keyMapChangeWriter))
		if cB.State == StateConflict {
			t.Errorf("op-identity guard: B must NOT be a false routedElsewhere Conflict from a raw-opcode coincidence; got %v (%s)", cB.State.Name(), cB.Note)
		}
		if cB.State != StatePartial {
			t.Errorf("B: present+routedElsewhere=false+toolPass report must be Partial; got %v (%s)", cB.State.Name(), cB.Note)
		}
		// A's own opcode is routed to a different op, so the guard also removes A
		// from routedVersions → A is not routed for itself → Partial (has report).
		if cA.State == StateConflict {
			t.Errorf("A must not be a Conflict from its own coincidental routing; got %v (%s)", cA.State.Name(), cA.Note)
		}
	})

	t.Run("matching-name-counts", func(t *testing.T) {
		// A routes 0x8B to A's OWN WEDDING_ACTION writer — the op is genuinely
		// routed in A. B (present, implemented, unrouted) is now routedElsewhere
		// → real template-wiring gap → Conflict.
		_, cB := cellsFor(build(weddingWriterA))
		if cB.State != StateConflict {
			t.Errorf("matching routed name: B must be a routedElsewhere template-wiring Conflict; got %v (%s)", cB.State.Name(), cB.Note)
		}
	})
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

// --- byte-fixture-without-report promotion (registry `packet:` link) -----------
//
// A packet with a committed golden byte-test (packet-audit:verify marker) + fresh
// evidence, but NO IDA-export audit report, must still promote to ✅ when the
// registry entry carries a `packet:` link. The report is confirmation, not a
// prerequisite: a golden byte-test is stronger proof than the static report diff.

// setItcRef mirrors the SET_ITC op (present, report-less clientbound writer).
func setItcRef() opEntryRef {
	return opEntryRef{Op: "SET_ITC", Dir: opregistry.DirClientbound, Opcode: 0x07E,
		FName: "CStage::OnSetITC", Packet: "field/clientbound/SetItc"}
}

func setItcInputs(t *testing.T) Inputs {
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "SET_ITC", Direction: opregistry.DirClientbound, Opcode: 0x07E,
		FName: "CStage::OnSetITC", Packet: "field/clientbound/SetItc", Provenance: "manual",
	})
	// Routed in the template so no wiring concern.
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x07E, opregistry.DirClientbound}: true}
	return in
}

func TestGradeByteFixtureNoReportPromotes(t *testing.T) {
	in := setItcInputs(t)
	pk := EvKey{Packet: "field/clientbound/SetItc", Version: "gms_v83"}
	in.Evidence[pk] = EvidenceStatus{Exists: true, Fresh: true, Address: "0x7774d1"}
	in.Markers[pk] = MarkerStatus{Found: true, Address: "0x7774d1"}
	c := gradeOpCell(in, setItcRef(), "gms_v83", false, map[string]bool{})
	if c.State != StateVerified {
		t.Fatalf("byte-fixture (marker+fresh evidence, no report) must be ✅; got %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeByteFixtureNoReportStaleEvidenceIncomplete(t *testing.T) {
	in := setItcInputs(t)
	pk := EvKey{Packet: "field/clientbound/SetItc", Version: "gms_v83"}
	in.Evidence[pk] = EvidenceStatus{Exists: true, Fresh: false, Address: "0x7774d1", Note: "hash drift"}
	in.Markers[pk] = MarkerStatus{Found: true, Address: "0x7774d1"}
	c := gradeOpCell(in, setItcRef(), "gms_v83", false, map[string]bool{})
	if c.State != StateIncomplete {
		t.Fatalf("stale evidence must NOT promote; got %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeByteFixtureNoReportNoMarkerIncomplete(t *testing.T) {
	in := setItcInputs(t)
	// Fresh evidence but no marker → not byte-verified.
	pk := EvKey{Packet: "field/clientbound/SetItc", Version: "gms_v83"}
	in.Evidence[pk] = EvidenceStatus{Exists: true, Fresh: true, Address: "0x7774d1"}
	c := gradeOpCell(in, setItcRef(), "gms_v83", false, map[string]bool{})
	if c.State != StateIncomplete {
		t.Fatalf("no marker must stay incomplete; got %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeNoReportNoPacketLinkStaysIncomplete(t *testing.T) {
	// Regression guard: an op with NO report and NO `packet:` link grades exactly
	// as before — Incomplete "no audit report" — even if some evidence/marker
	// exists under a different key. This is the zero-blast-radius guarantee.
	in := baseInputs()
	in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
		Op: "SET_ITC", Direction: opregistry.DirClientbound, Opcode: 0x07E,
		FName: "CStage::OnSetITC", Provenance: "manual", // NO Packet field
	})
	in.Routed["gms_v83"] = map[RouteKey]bool{{0x07E, opregistry.DirClientbound}: true}
	in.Evidence[EvKey{"field/clientbound/SetItc", "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true}
	in.Markers[EvKey{"field/clientbound/SetItc", "gms_v83"}] = MarkerStatus{Found: true}
	ref := opEntryRef{Op: "SET_ITC", Dir: opregistry.DirClientbound, Opcode: 0x07E, FName: "CStage::OnSetITC"} // no Packet
	c := gradeOpCell(in, ref, "gms_v83", false, map[string]bool{})
	if c.State != StateIncomplete || c.Note != "no audit report" {
		t.Fatalf("no packet link must stay Incomplete/no audit report; got %v (%s)", c.State.Name(), c.Note)
	}
}

func TestGradeByteFixtureNoReportFamilyCapped(t *testing.T) {
	// A dispatcher family (capped) with a byte-fixture + fresh evidence but no
	// report caps at 🧩, never ✅ (single sub-handler proves only one arm).
	in := setItcInputs(t)
	in.Families = map[string]bool{"CStage::OnSetITC": true}
	pk := EvKey{Packet: "field/clientbound/SetItc", Version: "gms_v83"}
	in.Evidence[pk] = EvidenceStatus{Exists: true, Fresh: true}
	in.Markers[pk] = MarkerStatus{Found: true}
	c := gradeOpCell(in, setItcRef(), "gms_v83", false, map[string]bool{})
	if c.State != StateFamily {
		t.Fatalf("capped dispatcher must stay 🧩 even byte-fixtured; got %v (%s)", c.State.Name(), c.Note)
	}
}
