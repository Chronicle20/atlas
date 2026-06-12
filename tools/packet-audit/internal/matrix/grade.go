package matrix

import (
	"fmt"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

type routeKey struct {
	Opcode int
	Dir    opregistry.Direction
}

type evKey struct {
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
	Registry       opregistry.Registry
	Reports        map[string]map[string]LoadedReport // version -> WriterName -> report
	FNameToWriter  map[string]map[string]string       // version -> FName -> WriterName (built from Reports)
	Routed         map[string]map[routeKey]bool       // version -> routed (opcode, dir)
	RoutedAnywhere map[routeKey]bool                  // routed in any version's template
	Evidence       map[evKey]EvidenceStatus
	Tier1          map[string]bool // packet id -> tier-1
	Markers        map[evKey]MarkerStatus
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
	applicability  opregistry.Applicability
	routed         bool
	routedAnywhere bool
	report         LoadedReport
	hasReport      bool
	evidence       EvidenceStatus
	hasEvidence    bool
	marker         MarkerStatus
	tier1          bool
	opcode         int
	writerName     string
}

// gradeOpCell evaluates design §5 in precedence order for one op×version.
func gradeOpCell(in Inputs, ref opEntryRef, version string) Cell {
	app := in.Registry.Applicability(ref.Op, ref.Dir, version)
	routed := in.Routed[version][routeKey{ref.Opcode, ref.Dir}]
	rep, hasReport := findReport(in, ref, version)

	var pkt string
	var ev EvidenceStatus
	var hasEv bool
	var mk MarkerStatus
	var tier1 bool
	if hasReport {
		pkt = PacketID(rep)
		ev, hasEv = in.Evidence[evKey{pkt, version}]
		mk = in.Markers[evKey{pkt, version}]
		tier1 = in.Tier1[pkt] || rep.FlatInvalid
	}

	args := gradeArgs{
		applicability:  app,
		routed:         routed,
		routedAnywhere: in.RoutedAnywhere[routeKey{ref.Opcode, ref.Dir}],
		report:         rep,
		hasReport:      hasReport,
		evidence:       ev,
		hasEvidence:    hasEv,
		marker:         mk,
		tier1:          tier1,
		opcode:         ref.Opcode,
		writerName:     rep.WriterName,
	}
	return gradeCore(args)
}

// gradeCore implements design §5 rules given fully-resolved gradeArgs.
func gradeCore(a gradeArgs) Cell {
	switch a.applicability {
	case opregistry.Unknown:
		return Cell{State: StateIncomplete, Note: "applicability unknown — no registry file for this version"}
	case opregistry.Absent:
		if a.routed {
			return Cell{State: StateConflict, Note: fmt.Sprintf("registry says absent but template routes opcode 0x%03X", a.opcode)}
		}
		if a.hasReport {
			return Cell{State: StateConflict, Note: "registry says absent but an Atlas audit report exists (" + a.writerName + ")"}
		}
		return Cell{State: StateNA}
	}

	// Present from here on.
	if !a.routed && a.routedAnywhere {
		return Cell{State: StateConflict, Note: "op present in client and routed in another version's template, but unrouted here (template coverage gap)"}
	}
	if !a.hasReport {
		return Cell{State: StateIncomplete, Note: "no audit report"}
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
