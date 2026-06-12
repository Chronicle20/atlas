package matrix

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// baseFName strips the per-case suffix: "CWvsContext::OnFriendResult#Invite"
// -> "CWvsContext::OnFriendResult".
func baseFName(idaName string) string {
	if i := strings.Index(idaName, "#"); i >= 0 {
		return idaName[:i]
	}
	return idaName
}

// Build joins all inputs into the Matrix. versionKeys fixes column order.
func Build(in Inputs, versionKeys []string) Matrix {
	// Index FName -> writers per version (a dispatcher FName may map to many
	// per-case writers; the op row takes the WORST graded cell of them).
	fnameWriters := map[string]map[string][]string{}
	for vk, reps := range in.Reports {
		fnameWriters[vk] = map[string][]string{}
		for wn, r := range reps {
			f := baseFName(r.IDAName)
			fnameWriters[vk][f] = append(fnameWriters[vk][f], wn)
		}
		for f := range fnameWriters[vk] {
			sort.Strings(fnameWriters[vk][f])
		}
	}

	usedWriters := map[string]map[string]bool{} // version -> writer consumed by an op row
	for _, vk := range versionKeys {
		usedWriters[vk] = map[string]bool{}
	}

	var rows []MatrixRow
	for _, od := range in.Registry.AllOps() {
		row := MatrixRow{Kind: RowOp, Op: od.Op, Direction: od.Dir, Cells: map[string]Cell{}}
		for _, vk := range versionKeys {
			ref := opEntryRef{Op: od.Op, Dir: od.Dir}
			if vf, ok := in.Registry.Versions[vk]; ok {
				if e, ok := vf.Lookup(od.Op, od.Dir); ok {
					ref.Opcode, ref.FName = e.Opcode, e.FName
				} else if e, ok := lookupAnyVersion(in.Registry, od.Op, od.Dir); ok {
					ref.Opcode, ref.FName = e.Opcode, e.FName // opcode for routing checks of absent ops
				}
			} else if e, ok := lookupAnyVersion(in.Registry, od.Op, od.Dir); ok {
				ref.Opcode, ref.FName = e.Opcode, e.FName
			}
			cell := worstCandidateCell(in, fnameWriters, ref, vk, usedWriters)
			row.Cells[vk] = cell
		}
		// Tier + packet annotation from any version's report.
		row.Packet, row.Tier1 = rowPacketAndTier(in, fnameWriters, row, versionKeys)
		rows = append(rows, row)
	}

	// Sub-struct rows: reports never consumed by an op row.
	sub := map[string]MatrixRow{}
	for _, vk := range versionKeys {
		for wn, r := range in.Reports[vk] {
			if usedWriters[vk][wn] {
				continue
			}
			pkt := PacketID(r)
			mr, ok := sub[pkt]
			if !ok {
				mr = MatrixRow{Kind: RowSubStruct, Packet: pkt, Cells: map[string]Cell{}}
			}
			mr.Tier1 = mr.Tier1 || in.Tier1[pkt] || r.FlatInvalid
			mr.Cells[vk] = gradeSubStructCell(in, r, pkt, vk)
			sub[pkt] = mr
		}
	}
	var subKeys []string
	for k := range sub {
		subKeys = append(subKeys, k)
	}
	sort.Strings(subKeys)
	for _, k := range subKeys {
		mr := sub[k]
		for _, vk := range versionKeys { // fill gaps so columns align
			if _, ok := mr.Cells[vk]; !ok {
				mr.Cells[vk] = Cell{State: StateIncomplete, Note: "no audit report"}
			}
		}
		rows = append(rows, mr)
	}
	return Matrix{Rows: rows}
}

func lookupAnyVersion(r opregistry.Registry, op string, dir opregistry.Direction) (opregistry.Entry, bool) {
	var vks []string
	for vk := range r.Versions {
		vks = append(vks, vk)
	}
	sort.Strings(vks)
	for _, vk := range vks {
		if e, ok := r.Versions[vk].Lookup(op, dir); ok {
			return e, true
		}
	}
	return opregistry.Entry{}, false
}

// worstCandidateCell grades each writer candidate for the op's FName and keeps
// the worst (max State); marks candidates as consumed by op rows.
// If two DIFFERENT writers carry the identical full IDAName (no #case suffix),
// that is a genuine duplicate claim — return StateConflict immediately.
func worstCandidateCell(in Inputs, fw map[string]map[string][]string, ref opEntryRef, vk string, used map[string]map[string]bool) Cell {
	writers := fw[vk][ref.FName]
	if len(writers) == 0 {
		// No candidates: grade without a report.
		return gradeOpCell(in, ref, vk)
	}

	// Check for duplicate-claim: multiple writers with the exact same full IDAName
	// (not just sharing a baseFName via #case suffixes).
	if len(writers) > 1 {
		// Count how many writers have the exact baseFName (no suffix) as their full IDAName.
		var exactMatches []string
		for _, wn := range writers {
			r := in.Reports[vk][wn]
			if r.IDAName == ref.FName {
				exactMatches = append(exactMatches, wn)
			}
		}
		if len(exactMatches) > 1 {
			return Cell{State: StateConflict, Note: fmt.Sprintf("two Atlas structs claim %s: %s, %s",
				ref.FName, exactMatches[0], exactMatches[1])}
		}
	}

	worst := Cell{State: StateNA, Note: ""}
	first := true
	for _, wn := range writers {
		used[vk][wn] = true
		// Build a single-entry FNameToWriter for this specific candidate.
		singleFName := map[string]map[string]string{vk: {ref.FName: wn}}
		inCopy := in
		inCopy.FNameToWriter = singleFName
		c := gradeOpCell(inCopy, ref, vk)
		if first || c.State > worst.State {
			worst, first = c, false
		}
	}
	return worst
}

// gradeSubStructCell grades a sub-struct report (no registry op — no
// applicability/routing logic applies). Uses gradeCore directly with
// applicability=Present, routed/routedAnywhere=false.
func gradeSubStructCell(in Inputs, r LoadedReport, pkt, vk string) Cell {
	ev, hasEv := in.Evidence[evKey{pkt, vk}]
	mk := in.Markers[evKey{pkt, vk}]
	tier1 := in.Tier1[pkt] || r.FlatInvalid

	args := gradeArgs{
		applicability:  opregistry.Present,
		routed:         true, // present + not routing-checked (sub-structs have no opcode)
		routedAnywhere: false,
		report:         r,
		hasReport:      true,
		evidence:       ev,
		hasEvidence:    hasEv,
		marker:         mk,
		tier1:          tier1,
		opcode:         -1,
		dir:            "",
		writerName:     r.WriterName,
	}
	return gradeCore(args)
}

func rowPacketAndTier(in Inputs, fw map[string]map[string][]string, row MatrixRow, versionKeys []string) (string, bool) {
	for _, vk := range versionKeys {
		if vf, ok := in.Registry.Versions[vk]; ok {
			if e, ok := vf.Lookup(row.Op, row.Direction); ok {
				for _, wn := range fw[vk][e.FName] {
					r := in.Reports[vk][wn]
					pkt := PacketID(r)
					return pkt, in.Tier1[pkt] || r.FlatInvalid
				}
			}
		}
	}
	return "", false
}
