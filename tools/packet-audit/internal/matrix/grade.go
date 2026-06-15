package matrix

import (
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// RouteKey is an (opcode, direction) pair used to record which ops a tenant
// template routes. Exported so cmd/matrix.go can build Inputs.
type RouteKey struct {
	Opcode int
	Dir    opregistry.Direction
}

// EvKey is the (packet-id, version) pair that keys evidence and marker maps.
// Exported so cmd/matrix.go can build Inputs.
type EvKey struct {
	Packet  string // "buddy/clientbound/Invite"
	Version string // "gms_v83"
}

// EvidenceStatus is the matrix-facing summary of one evidence record
// (loader lands in Phase 2; until then the map is empty).
type EvidenceStatus struct {
	Exists  bool
	Fresh   bool   // decompile_sha256 matches current export
	Address string // pinned ida address ("0x...")
	Note    string // degradation reason when !Fresh ("hash drift", "citation unresolvable")
}

// MarkerStatus is the matrix-facing summary of byte-test linkage
// (scanner lands in Phase 3; until then the map is empty).
type MarkerStatus struct {
	Found   bool
	Address string
}

// Inputs is everything grading consumes. All maps may be empty; rules that
// depend on them simply never fire.
type Inputs struct {
	Registry      opregistry.Registry
	Reports       map[string]map[string]LoadedReport // version -> WriterName -> report
	FNameToWriter map[string]map[string]string       // version -> FName -> WriterName (built from Reports)
	Routed        map[string]map[RouteKey]bool       // version -> routed (opcode, dir)
	// RoutedNames carries the writer/handler NAME a template routes each
	// (opcode, dir) to. Used to make the cross-version routedElsewhere signal
	// op-identity-aware: an opcode coincidentally occupied by a DIFFERENT op in
	// another version must not count as routing this op (design §10.1: avoid
	// raw-opcode-coincidence false conflicts). May be empty; when a key is
	// absent the routedElsewhere check falls back to opcode-occupancy.
	RoutedNames map[string]map[RouteKey]string
	Evidence      map[EvKey]EvidenceStatus
	Tier1         map[string]bool // packet id -> tier-1
	Markers       map[EvKey]MarkerStatus
	// Families is the set of base FNames that are mode-prefix DISPATCHERS
	// (one opcode, a leading mode byte switching to many sub-handlers with
	// distinct bodies). An op whose registry FName is in this set is capped at
	// StateFamily and can never reach ✅ on a single sub-handler's fixture.
	// Empty map = no family capping (every op grades normally).
	Families map[string]bool
}

// opEntryRef carries the union-row identity being graded for one version.
type opEntryRef struct {
	Op     string
	Dir    opregistry.Direction
	Opcode int
	FName  string
}

// gradeArgs is the resolved, version-specific input to the core grading logic.
// Using a struct avoids aliasing bugs when callers (worstCandidateCell,
// gradeSubStructCell) build per-candidate inputs from shared Inputs state.
type gradeArgs struct {
	applicability                 opregistry.Applicability
	routed                        bool
	routedElsewhere               bool // true when the same op is routed in at least one OTHER version (by that version's own opcode)
	report                        LoadedReport
	hasReport                     bool
	evidence                      EvidenceStatus
	hasEvidence                   bool
	marker                        MarkerStatus
	tier1                         bool
	opcode                        int
	writerName                    string
	reportFnameClaimedByPresentOp bool // true when a PRESENT op in the same version shares this report's base FName
	family                        bool // true when the op's FName is a mode-prefix dispatcher (cap at StateFamily)
}

// gradeOpCell evaluates design §5 in precedence order for one op×version.
// routedElsewhere must be pre-computed by the caller (Build) using per-version
// opcode resolution; it is true when the op is routed in at least one other
// version's template by that version's own opcode for the op.
// presentFnames is the set of FNames belonging to PRESENT ops in this version;
// when an absent op's report fname appears in this set, the absent-report
// conflict is suppressed (the report belongs to the present op, not this absent variant).
func gradeOpCell(in Inputs, ref opEntryRef, version string, routedElsewhere bool, presentFnames map[string]bool) Cell {
	app := in.Registry.Applicability(ref.Op, ref.Dir, version)
	routed := in.Routed[version][RouteKey{ref.Opcode, ref.Dir}]
	rep, hasReport := findReport(in, ref, version)

	var pkt string
	var ev EvidenceStatus
	var hasEv bool
	var mk MarkerStatus
	var tier1 bool
	if hasReport {
		pkt = PacketID(rep)
		ev, hasEv = in.Evidence[EvKey{pkt, version}]
		mk = in.Markers[EvKey{pkt, version}]
		tier1 = in.Tier1[pkt] || rep.FlatInvalid
	}

	// Determine whether the report's base FName is already claimed by a
	// PRESENT op in this version. If so, the absent-report conflict is
	// suppressed: the report belongs to the present op's variant, not here.
	reportFnameClaimedByPresentOp := false
	if hasReport && presentFnames != nil {
		reportFnameClaimedByPresentOp = presentFnames[baseFName(rep.IDAName)]
	}

	args := gradeArgs{
		applicability:                 app,
		routed:                        routed,
		routedElsewhere:               routedElsewhere,
		report:                        rep,
		hasReport:                     hasReport,
		evidence:                      ev,
		hasEvidence:                   hasEv,
		marker:                        mk,
		tier1:                         tier1,
		opcode:                        ref.Opcode,
		writerName:                    rep.WriterName,
		reportFnameClaimedByPresentOp: reportFnameClaimedByPresentOp,
		family:                        in.Families[baseFName(ref.FName)],
	}
	return gradeCore(args)
}

// familyNote explains why a dispatcher op is capped at StateFamily rather than
// promoted to ✅: the fixture proves the leading mode byte and the one fixtured
// sub-handler, but the remaining mode arms are neither implemented nor verified.
const familyNote = "mode-prefix dispatcher: leading mode byte + the fixtured sub-handler verified, but the remaining mode arms are unverified (see docs/packets/families.yaml)"

// gradeCore implements design §5 rules given fully-resolved gradeArgs.
func gradeCore(a gradeArgs) Cell {
	switch a.applicability {
	case opregistry.Unknown:
		return Cell{State: StateIncomplete, Note: "applicability unknown — no registry file for this version"}
	case opregistry.Absent:
		// Only raise the absent-report conflict if the report's fname is NOT
		// already claimed by a present op in this version. When a present op in
		// this version shares the same fname (e.g. a version-specific variant of
		// the same dispatcher), the report belongs to that present op and the
		// absent entry is simply not applicable here.
		if a.hasReport && reportResolved(a.report) && !a.reportFnameClaimedByPresentOp {
			return Cell{State: StateConflict, Note: "registry says absent but an Atlas audit report exists (" + a.writerName + ")"}
		}
		return Cell{State: StateNA}
	}

	// Present from here on.
	if !a.hasReport {
		return Cell{State: StateIncomplete, Note: "no audit report"}
	}
	// Atlas implements this op in this version (a report exists) but this
	// version's tenant template does not route its opcode, while another
	// version's template does — a real template-wiring gap (design §10.1
	// template leg), not mere absence.
	if !a.routed && a.routedElsewhere {
		return Cell{State: StateConflict, Note: "Atlas implements this op (audit report present) but this version's template does not route its opcode, though another version's does (template-wiring gap)"}
	}

	if a.hasEvidence && !a.evidence.Fresh {
		note := a.evidence.Note
		if note == "" {
			note = "evidence stale (decompile hash drift)"
		}
		return Cell{State: StateIncomplete, Note: note}
	}

	toolPass := a.report.Verdict == diff.VerdictMatch && !a.report.FlatInvalid

	if a.tier1 {
		// Diff verdict is advisory; only a linked byte-fixture promotes.
		if a.marker.Found && a.hasEvidence && a.evidence.Fresh {
			if a.family {
				return Cell{State: StateFamily, Note: familyNote}
			}
			return Cell{State: StateVerified}
		}
		if a.marker.Found {
			return Cell{State: StateIncomplete, Note: "byte-test marker present but no fresh evidence record"}
		}
		if toolPass || (a.hasEvidence && a.evidence.Fresh) {
			return Cell{State: StatePartial, Note: "tier-1: needs byte-fixture test to verify"}
		}
		return Cell{State: StateIncomplete, Note: "tier-1 without fixture; verdict " + a.report.Verdict.Symbol()}
	}

	// Tier 0.
	if toolPass && a.marker.Found {
		if a.family {
			return Cell{State: StateFamily, Note: familyNote}
		}
		return Cell{State: StateVerified}
	}
	if toolPass {
		return Cell{State: StatePartial, Note: "tool ✅ without byte-test"}
	}
	if a.hasEvidence && a.evidence.Fresh {
		return Cell{State: StatePartial, Note: "evidence-pinned deferral"}
	}
	return Cell{State: StateIncomplete, Note: "verdict " + a.report.Verdict.Symbol()}
}

// findReport joins a registry op to its audit report via FName -> WriterName.
func findReport(in Inputs, ref opEntryRef, version string) (LoadedReport, bool) {
	wn, ok := in.FNameToWriter[version][ref.FName]
	if !ok {
		return LoadedReport{}, false
	}
	r, ok := in.Reports[version][wn]
	return r, ok
}

// reportResolved returns true when the report carries a real IDA address
// (not an explicit placeholder). An unresolved report (Address ""/"ABSENT"/"0x0")
// means the IDB decompile did not locate the function, so the report does not
// constitute Atlas claiming ownership of that version's binary — it should not
// trigger the absent-branch conflict.
func reportResolved(r LoadedReport) bool {
	a := r.Address
	return a != "" && a != "ABSENT" && a != "0x0"
}
