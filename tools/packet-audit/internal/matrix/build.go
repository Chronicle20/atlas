package matrix

import (
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

	// Pre-compute per-version the set of base FNames that belong to PRESENT ops.
	// Used to suppress the absent-report conflict when a present op in the same
	// version already claims the report's fname (FIX B: design §5 absent branch).
	presentFnames := map[string]map[string]bool{}
	for _, vk := range versionKeys {
		presentFnames[vk] = map[string]bool{}
		if vf, ok := in.Registry.Versions[vk]; ok {
			for _, e := range vf.Entries {
				if e.FName != "" {
					presentFnames[vk][e.FName] = true
				}
			}
		}
	}

	usedWriters := map[string]map[string]bool{} // version -> writer consumed by an op row
	for _, vk := range versionKeys {
		usedWriters[vk] = map[string]bool{}
	}

	var rows []MatrixRow
	for _, od := range in.Registry.AllOps() {
		row := MatrixRow{Kind: RowOp, Op: od.Op, Direction: od.Dir, Cells: map[string]Cell{}}

		// Pre-compute which versions have this op PRESENT and ROUTED by that
		// version's own opcode. This is the per-packet routing set used to
		// compute routedElsewhere without false conflicts from raw-opcode
		// coincidences across versions.
		routedVersions := map[string]bool{}
		for _, vk := range versionKeys {
			e, ok := lookupVersion(in.Registry, od.Op, od.Dir, vk)
			if !ok {
				continue // op absent in this version → not routed here
			}
			if in.Routed[vk][RouteKey{e.Opcode, od.Dir}] {
				routedVersions[vk] = true
			}
		}

		for _, vk := range versionKeys {
			ref := opEntryRef{Op: od.Op, Dir: od.Dir}
			// Prefer this version's registry entry; fall back to any version so
			// absent ops still get an opcode for routing-conflict checks.
			if e, ok := lookupVersion(in.Registry, od.Op, od.Dir, vk); ok {
				ref.Opcode, ref.FName = e.Opcode, e.FName
			} else if e, ok := lookupAnyVersion(in.Registry, od.Op, od.Dir); ok {
				ref.Opcode, ref.FName = e.Opcode, e.FName
			}
			// routedElsewhere: the op is routed in at least one OTHER version's
			// template by that version's own opcode.
			routedElsewhere := false
			for ovk := range routedVersions {
				if ovk != vk {
					routedElsewhere = true
					break
				}
			}
			cell := worstCandidateCell(in, fnameWriters, ref, vk, usedWriters, routedElsewhere, presentFnames[vk])
			// Set the per-version opcode on the cell: the registry opcode from
			// this specific version if the op is present there, else -1.
			if e, ok := lookupVersion(in.Registry, od.Op, od.Dir, vk); ok {
				cell.Opcode = e.Opcode
			} else {
				cell.Opcode = -1
			}
			row.Cells[vk] = cell
		}
		// Tier + packet annotation from any version's report.
		row.Packet, row.Tier1 = rowPacketAndTier(in, fnameWriters, row, versionKeys)
		// Collect distinct base FNames across versions where the op is present.
		row.FNames = rowFNames(in.Registry, od.Op, od.Dir, versionKeys)
		rows = append(rows, row)
	}

	// Sort op rows by baseline opcode ascending; baseline = opcode from the
	// first version (in versionKeys order) that has the op present.
	// Tie-break by op name ascending.
	sort.SliceStable(rows, func(i, j int) bool {
		oi := baselineOpcode(rows[i], versionKeys)
		oj := baselineOpcode(rows[j], versionKeys)
		if oi != oj {
			return oi < oj
		}
		return rows[i].Op < rows[j].Op
	})

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
			c := gradeSubStructCell(in, r, pkt, vk)
			c.Opcode = -1 // sub-struct cells always have no opcode
			mr.Cells[vk] = c
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
				mr.Cells[vk] = Cell{State: StateIncomplete, Note: "no audit report", Opcode: -1}
			}
		}
		rows = append(rows, mr)
	}
	return Matrix{Rows: rows}
}

// baselineOpcode returns the opcode from the first version (in versionKeys
// order) where the op row has a non-negative opcode, or math.MaxInt32 as a
// fallback so rows with no present version sort last.
func baselineOpcode(row MatrixRow, versionKeys []string) int {
	for _, vk := range versionKeys {
		if c, ok := row.Cells[vk]; ok && c.Opcode >= 0 {
			return c.Opcode
		}
	}
	return 1<<31 - 1 // sort absent-everywhere rows last
}

// rowFNames collects the distinct base FNames (plus FNameAlts) across all
// versions where the op is present in the registry. Empty FNames (UNNAMED_R
// rows) are dropped. Result is sorted and deduplicated.
func rowFNames(reg opregistry.Registry, op string, dir opregistry.Direction, versionKeys []string) []string {
	seen := map[string]bool{}
	for _, vk := range versionKeys {
		e, ok := lookupVersion(reg, op, dir, vk)
		if !ok {
			continue
		}
		if e.FName != "" {
			seen[e.FName] = true
		}
		for _, alt := range e.FNameAlts {
			if alt != "" {
				seen[alt] = true
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// lookupVersion looks up op+dir in a specific version's file, if it exists.
func lookupVersion(r opregistry.Registry, op string, dir opregistry.Direction, vk string) (opregistry.Entry, bool) {
	if vf, ok := r.Versions[vk]; ok {
		return vf.Lookup(op, dir)
	}
	return opregistry.Entry{}, false
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
// the worst (by severity()); marks candidates as consumed by op rows.
// When multiple writers share a base FName (a legitimate client-function demux
// such as CUser::OnEffect or CLogin::OnViewAllCharResult), the op row grades
// worst-of across all candidates. No conflict is raised for shared base FNames
// regardless of whether the full IDAName includes a #case suffix or not —
// demux families are expected to share a dispatcher name.
// routedElsewhere is pre-computed by Build (per-op, per-version) and threaded
// through to gradeOpCell to implement the per-packet cross-version routing rule.
// presentFnames is the set of FNames belonging to PRESENT ops in this version;
// it is forwarded to gradeOpCell to suppress false absent-report conflicts.
func worstCandidateCell(in Inputs, fw map[string]map[string][]string, ref opEntryRef, vk string, used map[string]map[string]bool, routedElsewhere bool, presentFnames map[string]bool) Cell {
	writers := fw[vk][ref.FName]
	if len(writers) == 0 {
		// No candidates: grade without a report; use an empty FNameToWriter for
		// this version so Build always derives its own index rather than leaking
		// the caller-supplied map through.
		inCopy := in
		inCopy.FNameToWriter = map[string]map[string]string{vk: {}}
		return gradeOpCell(inCopy, ref, vk, routedElsewhere, presentFnames)
	}

	worst := Cell{State: StateNA, Note: ""}
	first := true
	for _, wn := range writers {
		used[vk][wn] = true
		// Build a single-entry FNameToWriter for this specific candidate.
		singleFName := map[string]map[string]string{vk: {ref.FName: wn}}
		inCopy := in
		inCopy.FNameToWriter = singleFName
		c := gradeOpCell(inCopy, ref, vk, routedElsewhere, presentFnames)
		if first || severity(c.State) > severity(worst.State) {
			worst, first = c, false
		}
	}
	return worst
}

// gradeSubStructCell grades a sub-struct report (no registry op — no
// applicability/routing logic applies). Uses gradeCore directly with
// applicability=Present, routed=true, routedElsewhere=false (sub-structs have
// no opcode so the cross-version routing signal never fires).
func gradeSubStructCell(in Inputs, r LoadedReport, pkt, vk string) Cell {
	ev, hasEv := in.Evidence[EvKey{pkt, vk}]
	mk := in.Markers[EvKey{pkt, vk}]
	tier1 := in.Tier1[pkt] || r.FlatInvalid

	args := gradeArgs{
		applicability:   opregistry.Present,
		routed:          true, // present + not routing-checked (sub-structs have no opcode)
		routedElsewhere: false,
		report:          r,
		hasReport:       true,
		evidence:        ev,
		hasEvidence:     hasEv,
		marker:          mk,
		tier1:           tier1,
		opcode:          -1,
		writerName:      r.WriterName,
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
