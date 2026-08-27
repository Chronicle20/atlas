package matrix

import (
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// dispositionNote annotates a sub-struct cell graded n-a because the version
// deliberately disposes the sub-struct as version-absent (_unimplemented.json).
const dispositionNote = "disposition: version-absent (n-a, see _unimplemented.json)"

// contains reports whether s is present in xs.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// baseFName strips the per-case suffix: "CWvsContext::OnFriendResult#Invite"
// -> "CWvsContext::OnFriendResult".
func baseFName(idaName string) string {
	if i := strings.Index(idaName, "#"); i >= 0 {
		return idaName[:i]
	}
	return idaName
}

// opKey builds the (op, direction) map key used by legacyConsumedSiblingWriters.
func opKey(op string, dir opregistry.Direction) string {
	return op + "|" + string(dir)
}

// versionScopedOpKey builds the (op, direction, version) map key used by
// legacyConsumedSiblingWriters entries that must apply to a SINGLE version
// only, because the same fname collision recurs on other versions where the
// sibling has not yet been per-cell verified (see the NPC_TALK_MORE entry
// below).
func versionScopedOpKey(op string, dir opregistry.Direction, vk string) string {
	return opKey(op, dir) + "|" + vk
}

// siblingWritersFor resolves the sibling WriterNames that escape op-row
// consumption for (op, dir) in version vk: the union of the op-wide entry
// (applies to every version — NOTE_ACTION and USE_CASH_ITEM below) and any
// version-scoped entry keyed by versionScopedOpKey (applies to exactly the
// named version, leaving the same collision on every other version alone).
func siblingWritersFor(op string, dir opregistry.Direction, vk string) map[string]bool {
	global := legacyConsumedSiblingWriters[opKey(op, dir)]
	scoped := legacyConsumedSiblingWriters[versionScopedOpKey(op, dir, vk)]
	if len(global) == 0 {
		return scoped
	}
	if len(scoped) == 0 {
		return global
	}
	out := make(map[string]bool, len(global)+len(scoped))
	for wn := range global {
		out[wn] = true
	}
	for wn := range scoped {
		out[wn] = true
	}
	return out
}

// legacyConsumedSiblingWriters is an explicit, narrowly-scoped allowlist of
// (op, direction) -> the specific sibling WriterName(s) that get swallowed by
// the op row in a SUBSET of versions rather than graded independently
// everywhere (task-137 legacy NOTE_ACTION fix, see
// docs/tasks/task-137-note-item-consumption/design.md and
// .superpowers/sdd/task-17-legacy-diagnosis.md).
//
// Keys are built with either opKey(op, dir) — applies to every version where
// the fname collision recurs (NOTE_ACTION and USE_CASH_ITEM below, both
// per-cell-verified on every version they touch) — or versionScopedOpKey(op,
// dir, vk) — applies to exactly one named version, for a per-cell-verified
// sibling whose SAME collision also recurs on other versions that have not
// yet been individually verified (NPC_TALK_MORE below). siblingWritersFor
// resolves both forms for a given version.
//
// NOTE_ACTION/serverbound: on gms_v48/v61/v72/v79 the op's registry primary
// `fname` is CMemoListDlg::SetRet, which is ALSO the exact IDAName of the
// separately-reported, separately-fixtured NoteOperationDiscard sub-struct —
// so that writer gets consumed into the op row and skipped in the sub-struct
// pass on those four versions only. On gms_v83/84/87/95/jms_v185 the op's
// primary `fname` is a DIFFERENT writer (CCashShop::OnCashItemResLoadGiftDone,
// NoteOperationSend's IDAName; CMemoListDlg::SetRet is only listed as an
// fname_alt there), so NoteOperationDiscard is never consumed and grades
// independently — hence today it already reads ✅ on those five versions and
// ❌ ("no audit report") on the four legacy ones despite having its own
// pinned TIER1 marker + evidence + report on all nine.
//
// Listed by exact WriterName, not just by op: NOTE_ACTION's OWN primary
// writer changes identity across the version set too (NoteOperationSend's
// IDAName IS the modern primary fname on gms_v83/84/87/95/jms_v185, so it is
// ALSO consumed there — it is not a sibling being wrongly swallowed on those
// five versions, it is the op row's own writer).
//
// task-137 notesend-verify2: NoteOperationSend itself is now ALSO listed
// here, per-cell-verified across all nine versions (byte-golden fixture +
// pinned TIER1 evidence + audit report for every version, including a v79
// report generated for the first time and a v48 report corrected to cite the
// real send-site function, CCashShop::OnCashItemResLoadLockerDone — see
// operation_send_test.go TestOperationSendByteOutputAllVersions and
// docs/packets/evidence/*/note.serverbound.NoteOperationSend.yaml). Listing it
// here lets the sub-struct pass grade it from its OWN evidence on
// gms_v83/84/87/95/jms_v185 too (where it is consumed by the op row) instead
// of gap-filling it — mirroring exactly what NoteOperationDiscard already got
// for the v48/61/72/79 leg. A prior pass explicitly deferred adding this entry
// ("un-suppressing it flipped 5 cells that were not part of this fix's
// scope") specifically because those five cells had not yet been verified;
// now that they have (this pass), the deferral no longer applies.
//
// This is deliberately an explicit allowlist, NOT a structural rule (e.g.
// "any op with more than one distinct writer across versions", or "any op
// whose primary fname for one version is an fname_alt for another version of
// the same op"): both were tried and measured to also flip dozens of
// unrelated, already-correct cells (account/serverbound/RegisterPin and the
// entire buddy clientbound result family among them) whose sibling arms are
// legitimately meant to stay suppressed in most versions. A general rule
// capable of distinguishing NOTE_ACTION's case from those without also
// perturbing them does not exist as a cheap registry-topology check; if
// another op develops the same defect, add its specific sibling WriterName
// here explicitly after the same per-cell verification NOTE_ACTION got, not
// by broadening this predicate.
// USE_CASH_ITEM/serverbound: on gms_v72/v79 the op's registry primary
// `fname` is CWvsContext::SendConsumeCashItemUseRequest, which is ALSO the
// exact IDAName of several separately-reported, separately-fixtured
// per-case sub-struct writers (CashItemUseSongPlayer among them — the
// jukebox/song-player cash item, task-252) — so those writers get consumed
// into the USE_CASH_ITEM op row and skipped in the sub-struct pass on those
// two versions. On gms_v83/84/87/92/95/jms_v185 the op's primary `fname` is
// a DIFFERENT writer (CItemSpeakerDlg::_SendConsumeCashItemUseRequest), so
// the sub-struct writers are never consumed there and grade independently
// already (✅ on all six once individually verified).
//
// Only CashItemUseSongPlayer is listed here: it alone has its own pinned
// TIER1 evidence + byte-fixture marker + audit report for gms_v72 AND
// gms_v79 (task-252). Per the protectedWriters gate in Build, listing a
// sibling here only lets it escape the automatic skip WHEN its own
// gradeSubStructCell independently reaches StateVerified — it is not a
// force-promote. The other USE_CASH_ITEM siblings that share this same
// fname collision on v72/v79 (CashItemUseSuperMegaphone,
// CashItemUseMapleTV, CashItemUseMegaphone, CashItemUseTripleMegaphone)
// are NOT added here: fixing their v72/v79 cells is out of scope for
// task-252 and is left for whichever task next verifies them per-version.
// NPC_TALK_MORE/serverbound: on gms_v95 the op's registry primary `fname` is
// CScriptMan::OnAskSlideMenu (docs/packets/registry/gms_v95.yaml:2549-2554),
// which is exactly baseFName("CScriptMan::OnAskSlideMenu#AskSlideMenu") —
// the clientbound detail writer NpcAskSlideMenuConversationDetail's own
// IDAName — so the op row consumes it and the sub-struct pass skips it there.
// The sibling NpcSayImageConversationDetail escapes this same op row only
// because CScriptMan::OnSayImage is listed as an fname_alt rather than the
// primary; it is unaffected by this entry. NpcAskSlideMenuConversationDetail
// carries its own marker (conversation_test.go:25, ida=0x6dbe50), pinned
// evidence (docs/packets/evidence/gms_v95/npc.clientbound.NpcAskSlideMenuConversationDetail.yaml),
// and audit report on gms_v95. Per the protectedWriters gate in Build,
// listing it here only lets it escape the automatic skip WHEN its own
// gradeSubStructCell independently reaches StateVerified — it is not a
// force-promote.
//
// Version-scoped, unlike NOTE_ACTION and USE_CASH_ITEM above: the SAME
// primary-fname collision (CScriptMan::OnAskSlideMenu) recurs verbatim in
// the registry on gms_v83/84/87/92/jms_v185 too, and conversation_test.go
// carries markers for the sibling on v83 (line 23), v87 (24), v84 (87), and
// jms_v185 (200) as well — an op-wide (non-scoped) entry would therefore
// have un-suppressed all five of those cells in the same pass this task
// scoped to v95 alone. Scoping this entry to gms_v95 only, via
// versionScopedOpKey, defers v83/84/87/jms_v185 (each independently
// verifiable the same way) and v92 (no marker found at all) to whichever
// task next verifies them per-version — mirroring exactly the deferral
// discipline NOTE_ACTION's own history records above for NoteOperationSend.
var legacyConsumedSiblingWriters = map[string]map[string]bool{
	opKey("NOTE_ACTION", opregistry.DirServerbound): {
		"NoteOperationDiscard": true,
		"NoteOperationSend":    true,
	},
	opKey("USE_CASH_ITEM", opregistry.DirServerbound): {
		"CashItemUseSongPlayer": true,
	},
	versionScopedOpKey("NPC_TALK_MORE", opregistry.DirServerbound, "gms_v95"): {
		"NpcAskSlideMenuConversationDetail": true,
	},
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
	// protectedWriters marks a (version, writer) consumed by an op row where
	// the writer is explicitly listed in legacyConsumedSiblingWriters for that
	// op — the sub-struct pass only bypasses the op-row consumption skip for a
	// protected writer when its OWN pinned TIER1 evidence independently
	// verifies the cell (never on a lesser grade), so an op is unaffected here
	// unless its listed sibling writer is already fully verified on its own.
	protectedWriters := map[string]map[string]bool{}
	for _, vk := range versionKeys {
		usedWriters[vk] = map[string]bool{}
		protectedWriters[vk] = map[string]bool{}
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
			rk := RouteKey{e.Opcode, od.Dir}
			if !in.Routed[vk][rk] {
				continue
			}
			// Op-identity guard: when the template's name at this opcode is known
			// AND this version maps the op's FName to a specific writer (i.e. Atlas
			// implements the op here), require the routed name to be that writer.
			// This rejects raw-opcode coincidences where the opcode is occupied by a
			// DIFFERENT op in this version (e.g. a serverbound WEDDING_ACTION opcode
			// that happens to equal a CharacterKeyMapChange handler slot in v84),
			// which would otherwise fabricate a routedElsewhere → template-wiring
			// conflict for the unrelated versions. Falls back to opcode-occupancy
			// when either name is unknown, preserving prior behavior for every op
			// that does not have this collision.
			if rn, hasRN := in.RoutedNames[vk]; hasRN {
				if want, okW := fnameWriters[vk][e.FName]; okW && len(want) > 0 {
					if got := rn[rk]; got != "" && !contains(want, got) {
						continue // opcode routed to a different op — not this one
					}
				}
			}
			routedVersions[vk] = true
		}

		for _, vk := range versionKeys {
			ref := opEntryRef{Op: od.Op, Dir: od.Dir}
			// Prefer this version's registry entry; fall back to any version so
			// absent ops still get an opcode for routing-conflict checks.
			if e, ok := lookupVersion(in.Registry, od.Op, od.Dir, vk); ok {
				ref.Opcode, ref.FName, ref.Packet = e.Opcode, e.FName, e.Packet
			} else if e, ok := lookupAnyVersion(in.Registry, od.Op, od.Dir); ok {
				ref.Opcode, ref.FName, ref.Packet = e.Opcode, e.FName, e.Packet
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
			siblingWriters := siblingWritersFor(od.Op, od.Dir, vk)
			cell := worstCandidateCell(in, fnameWriters, ref, vk, usedWriters, protectedWriters, siblingWriters, routedElsewhere, presentFnames[vk])
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
			pkt := PacketID(r)
			if usedWriters[vk][wn] {
				if !protectedWriters[vk][wn] {
					continue
				}
				// Protected: wn is explicitly listed in
				// legacyConsumedSiblingWriters for the op that consumed it
				// here, because THIS version's op primary fname happens to
				// equal wn's own fname (see legacyConsumedSiblingWriters
				// comment above). Bypass the skip ONLY when wn's own
				// pinned TIER1 evidence independently verifies this cell —
				// never on a lesser grade, so a cell that would otherwise
				// gap-fill (or grade some non-verified state) keeps that
				// exact state/note.
				if gradeSubStructCell(in, r, pkt, vk).State != StateVerified {
					continue
				}
			}
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
				// A gap-filled sub-struct cell (this version has no audit report)
				// is n-a when the (packetID, version) is dispositioned as
				// version-absent in _unimplemented.json; otherwise it is an
				// un-audited gap (FR-4.1, task-169).
				if in.Unimplemented[vk][k] {
					mr.Cells[vk] = Cell{State: StateNA, Note: dispositionNote, Opcode: -1}
				} else {
					mr.Cells[vk] = Cell{State: StateIncomplete, Note: "no audit report", Opcode: -1}
				}
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
// siblingWriters is the set of WriterNames (if any) listed in
// legacyConsumedSiblingWriters for this op; a consumed candidate whose name is
// in this set is marked protected so the sub-struct pass may grade it from its
// own evidence instead of gap-filling it (see legacyConsumedSiblingWriters doc).
func worstCandidateCell(in Inputs, fw map[string]map[string][]string, ref opEntryRef, vk string, used map[string]map[string]bool, protected map[string]map[string]bool, siblingWriters map[string]bool, routedElsewhere bool, presentFnames map[string]bool) Cell {
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
		if siblingWriters[wn] {
			protected[vk][wn] = true
		}
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
	// A version that explicitly disposes this sub-struct as version-absent grades
	// n-a even if a stray report exists (FR-4.1, task-169). In practice a
	// dispositioned version has no report (the flip happens in the gap-fill
	// branch), so this is a defensive guard that keeps the two paths consistent.
	if in.Unimplemented[vk][pkt] {
		return Cell{State: StateNA, Note: dispositionNote}
	}
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
