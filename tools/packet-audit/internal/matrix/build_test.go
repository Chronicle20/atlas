package matrix

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// subCell finds a sub-struct row by packet id and returns its cell for vk.
func subCell(t *testing.T, m Matrix, pkt, vk string) Cell {
	t.Helper()
	for _, r := range m.Rows {
		if r.Kind == RowSubStruct && r.Packet == pkt {
			return r.Cells[vk]
		}
	}
	t.Fatalf("no sub-struct row for packet %q", pkt)
	return Cell{}
}

// A sub-struct row exists because v83 has a report; v48 has no report. Without
// a disposition, v48's gap-filled cell is Incomplete ("no audit report").
func TestSubStructUndispositionedIsIncomplete(t *testing.T) {
	in := baseInputs()
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"NpcSayImageConversationDetail": {
			WriterName: "NpcSayImageConversationDetail",
			IDAName:    "CScriptMan::OnSayImage#Detail",
			AtlasFile:  "libs/atlas-packet/npc/clientbound/say_image.go",
			Verdict:    diff.VerdictMatch,
		},
	}
	in.Reports["gms_v48"] = map[string]LoadedReport{}
	pkt := "npc/clientbound/NpcSayImageConversationDetail"

	m := Build(in, []string{"gms_v48", "gms_v83"})
	c := subCell(t, m, pkt, "gms_v48")
	if c.State != StateIncomplete {
		t.Fatalf("undispositioned sub-struct v48 = %v (%s); want incomplete", c.State.Name(), c.Note)
	}
}

// When (packet, version) is dispositioned in Unimplemented, the sub-struct's
// gap-filled cell grades n-a (StateNA) instead of Incomplete. (task-169 T2.1 / FR-4.1)
func TestSubStructDispositionedIsNA(t *testing.T) {
	in := baseInputs()
	in.Reports["gms_v83"] = map[string]LoadedReport{
		"NpcSayImageConversationDetail": {
			WriterName: "NpcSayImageConversationDetail",
			IDAName:    "CScriptMan::OnSayImage#Detail",
			AtlasFile:  "libs/atlas-packet/npc/clientbound/say_image.go",
			Verdict:    diff.VerdictMatch,
		},
	}
	in.Reports["gms_v48"] = map[string]LoadedReport{}
	pkt := "npc/clientbound/NpcSayImageConversationDetail"
	in.Unimplemented = map[string]map[string]bool{"gms_v48": {pkt: true}}

	m := Build(in, []string{"gms_v48", "gms_v83"})
	c := subCell(t, m, pkt, "gms_v48")
	if c.State != StateNA {
		t.Fatalf("dispositioned sub-struct v48 = %v (%s); want n-a", c.State.Name(), c.Note)
	}
	// The version WITH a report (v83) is unaffected by the disposition.
	if got := subCell(t, m, pkt, "gms_v83"); got.State == StateNA {
		t.Fatalf("v83 (has report) must not be n-a; got %v", got.State.Name())
	}
}

// ResolveUnimplemented: explicit `packet` paths and suffix-qualified fnames
// resolve; a bare base fname (dispatcher arm/sender disposition) does NOT — its
// base name collides with an implemented sibling struct's IDAName.
func TestResolveUnimplemented(t *testing.T) {
	idx := map[string]string{
		"CScriptMan::OnAskPet#AskPet":   "npc/clientbound/NpcAskPetConversationDetail",
		"CLogin::OnCheckPasswordResult": "login/clientbound/AuthSuccess",
	}
	refs := []UnimplementedRef{
		{Packet: "interaction/serverbound/InteractionOperationMerchantAddToBlackList"}, // explicit packet
		{FName: "CScriptMan::OnAskPet#AskPet"},                                         // suffix-qualified
		{FName: "CLogin::OnCheckPasswordResult"},                                       // bare base fname -> MUST NOT resolve
		{FName: "CField::OnFieldEffect"},                                               // bare, not in index
	}
	got := ResolveUnimplemented(refs, idx)
	if !got["interaction/serverbound/InteractionOperationMerchantAddToBlackList"] {
		t.Error("explicit packet path should resolve")
	}
	if !got["npc/clientbound/NpcAskPetConversationDetail"] {
		t.Error("suffix-qualified fname should resolve")
	}
	if got["login/clientbound/AuthSuccess"] {
		t.Error("bare base fname must NOT resolve (would downgrade an implemented sibling struct)")
	}
	if len(got) != 2 {
		t.Errorf("resolved set size = %d; want 2 (%v)", len(got), got)
	}
}

// TestAskSlideMenuEscapesNpcTalkMoreConsumption pins the
// legacyConsumedSiblingWriters entry for NPC_TALK_MORE/serverbound: on
// gms_v95 the op's registry primary fname (CScriptMan::OnAskSlideMenu) is
// exactly baseFName of NpcAskSlideMenuConversationDetail's own IDAName
// (CScriptMan::OnAskSlideMenu#AskSlideMenu), so the op row would otherwise
// consume and suppress the sibling. The protectedWriters gate must only let
// it escape when its OWN evidence independently reaches StateVerified —
// never as a force-promote.
//
// gms_v83 carries a second, independent registry entry for the same op with
// a DIFFERENT fname (not matching the sibling's IDAName), so its own report
// is never consumed there; this is what creates the sub-struct row at all
// and lets the gms_v95 column exercise the gap-fill path in cases 2 and 3.
func TestAskSlideMenuEscapesNpcTalkMoreConsumption(t *testing.T) {
	const pkt = "npc/clientbound/NpcAskSlideMenuConversationDetail"
	versionKeys := []string{"gms_v83", "gms_v95"}

	report := LoadedReport{
		WriterName: "NpcAskSlideMenuConversationDetail",
		IDAName:    "CScriptMan::OnAskSlideMenu#AskSlideMenu",
		AtlasFile:  "libs/atlas-packet/npc/clientbound/conversation.go",
		Verdict:    diff.VerdictMatch,
	}

	newInputs := func() Inputs {
		in := baseInputs()
		in.Registry.Versions["gms_v95"] = vfWith(t, opregistry.Entry{
			Op: "NPC_TALK_MORE", Direction: opregistry.DirServerbound, Opcode: 0x41,
			FName: "CScriptMan::OnAskSlideMenu", Provenance: "ida-discovered",
		})
		in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
			Op: "NPC_TALK_MORE", Direction: opregistry.DirServerbound, Opcode: 0x41,
			FName: "CScriptMan::OnUnrelatedTalkMoreV83", Provenance: "ida-discovered",
		})
		in.Reports["gms_v83"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
		in.Tier1[pkt] = true
		return in
	}

	tests := []struct {
		name      string
		setup     func(in *Inputs)
		wantState State
		wantNote  string
		checkOp   bool
		wantOp    int
	}{
		{
			name: "verified sibling escapes",
			setup: func(in *Inputs) {
				in.Reports["gms_v95"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
				in.Markers[EvKey{pkt, "gms_v95"}] = MarkerStatus{Found: true, Address: "0x6dbe50"}
				in.Evidence[EvKey{pkt, "gms_v95"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0x6dbe50"}
			},
			wantState: StateVerified,
			wantNote:  "",
			checkOp:   true,
			wantOp:    -1,
		},
		{
			name: "unverified sibling stays suppressed",
			setup: func(in *Inputs) {
				in.Reports["gms_v95"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
				// No marker, no evidence: gradeSubStructCell cannot reach
				// StateVerified, so the protectedWriters bypass must not fire.
			},
			wantState: StateIncomplete,
			wantNote:  "no audit report",
		},
		{
			name: "stale evidence stays suppressed",
			setup: func(in *Inputs) {
				in.Reports["gms_v95"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
				in.Markers[EvKey{pkt, "gms_v95"}] = MarkerStatus{Found: true, Address: "0x6dbe50"}
				in.Evidence[EvKey{pkt, "gms_v95"}] = EvidenceStatus{Exists: true, Fresh: false}
			},
			wantState: StateIncomplete,
			wantNote:  "no audit report",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := newInputs()
			tc.setup(&in)

			m := Build(in, versionKeys)
			c := subCell(t, m, pkt, "gms_v95")
			if c.State != tc.wantState || c.Note != tc.wantNote || (tc.checkOp && c.Opcode != tc.wantOp) {
				t.Fatalf("gms_v95 cell = %v (%q, opcode=%d); want %v, %q", c.State.Name(), c.Note, c.Opcode, tc.wantState.Name(), tc.wantNote)
			}
		})
	}
}

// TestGmsV95FnamePromotionEscapesSubStructConsumption pins the four
// versionScopedOpKey(..., "gms_v95") legacyConsumedSiblingWriters entries
// added for task-146's fname-promotion regression fix (CHANGE_MAP/serverbound
// FieldChange, NPC_TALK/clientbound NpcNpcConversation, NPC_TALK/serverbound
// NpcStartConversation, CHANGE_MAP_SPECIAL/serverbound PortalScript): each
// promoted op's registry primary fname now equals baseFName of the sibling
// sub-struct's own IDAName, so the op row's worstCandidateCell consumes the
// sibling as a used candidate. The entries must let each sibling escape and
// grade from its own evidence ONLY on gms_v95 — the version each entry is
// scoped to.
//
// The load-bearing case is the negative one: the SAME fname collision is
// constructed here on gms_v83 (a version the fix's entries do NOT name), with
// an equally-verified report, marker, and evidence. If versionScopedOpKey's
// version key were dropped (an unscoped, op-wide entry, or one copy-pasted to
// the wrong version), this gms_v83 case would also escape consumption and
// flip from suppressed to verified — exactly the regression the review found
// task 5's precedent (NPC_TALK_MORE) already guarded against, and which the
// build.go comment for this fix records was measured to actually occur on
// gms_v83/v84/v87/jms_v185 for these four op fnames.
func TestGmsV95FnamePromotionEscapesSubStructConsumption(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		dir     opregistry.Direction
		writer  string
		idaName string
		pkt     string
	}{
		{
			name:    "FieldChange",
			op:      "CHANGE_MAP",
			dir:     opregistry.DirServerbound,
			writer:  "FieldChange",
			idaName: "CField::SendTransferFieldRequest",
			pkt:     "field/serverbound/FieldChange",
		},
		{
			name:    "NpcNpcConversation",
			op:      "NPC_TALK",
			dir:     opregistry.DirClientbound,
			writer:  "NpcNpcConversation",
			idaName: "CScriptMan::OnScriptMessage",
			pkt:     "npc/clientbound/NpcNpcConversation",
		},
		{
			name:    "NpcStartConversation",
			op:      "NPC_TALK",
			dir:     opregistry.DirServerbound,
			writer:  "NpcStartConversation",
			idaName: "CUserLocal::TalkToNpc",
			pkt:     "npc/serverbound/NpcStartConversation",
		},
		{
			name:    "PortalScript",
			op:      "CHANGE_MAP_SPECIAL",
			dir:     opregistry.DirServerbound,
			writer:  "PortalScript",
			idaName: "CUserLocal::CheckPortal_Collision",
			pkt:     "portal/serverbound/PortalScript",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			versionKeys := []string{"gms_v83", "gms_v95"}

			report := LoadedReport{
				WriterName: tc.writer,
				IDAName:    tc.idaName,
				AtlasFile:  "libs/atlas-packet/" + tc.pkt[:strings.LastIndex(tc.pkt, "/")] + "/x.go",
				Verdict:    diff.VerdictMatch,
			}

			newInputs := func() Inputs {
				in := baseInputs()
				// Same fname collision constructed on BOTH versions: the op's
				// registry primary fname equals tc.idaName (== tc.writer's own
				// IDAName) on both, so the sibling is consumed by the op row
				// on both unless a scoped entry lets it escape.
				in.Registry.Versions["gms_v95"] = vfWith(t, opregistry.Entry{
					Op: tc.op, Direction: tc.dir, Opcode: 0x50, FName: tc.idaName, Provenance: "ida-discovered",
				})
				in.Registry.Versions["gms_v83"] = vfWith(t, opregistry.Entry{
					Op: tc.op, Direction: tc.dir, Opcode: 0x50, FName: tc.idaName, Provenance: "ida-discovered",
				})
				in.Reports["gms_v95"] = map[string]LoadedReport{tc.writer: report}
				in.Reports["gms_v83"] = map[string]LoadedReport{tc.writer: report}
				in.Tier1[tc.pkt] = true
				// Fully verified evidence on BOTH versions: if the negative
				// case fails, it fails because of the entry's scoping, not
				// because gms_v83's own evidence is weaker.
				in.Markers[EvKey{tc.pkt, "gms_v95"}] = MarkerStatus{Found: true, Address: "0x1000"}
				in.Evidence[EvKey{tc.pkt, "gms_v95"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0x1000"}
				in.Markers[EvKey{tc.pkt, "gms_v83"}] = MarkerStatus{Found: true, Address: "0x1000"}
				in.Evidence[EvKey{tc.pkt, "gms_v83"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0x1000"}
				return in
			}

			in := newInputs()
			m := Build(in, versionKeys)

			v95 := subCell(t, m, tc.pkt, "gms_v95")
			if v95.State != StateVerified || v95.Note != "" || v95.Opcode != -1 {
				t.Fatalf("gms_v95 cell = %v (%q, opcode=%d); want StateVerified, \"\", -1 (entry should let %s escape consumption)", v95.State.Name(), v95.Note, v95.Opcode, tc.writer)
			}

			v83 := subCell(t, m, tc.pkt, "gms_v83")
			if v83.State != StateIncomplete || v83.Note != "no audit report" {
				t.Fatalf("gms_v83 cell = %v (%q); want StateIncomplete, \"no audit report\" (the gms_v95-scoped entry must NOT apply to gms_v83's identical collision)", v83.State.Name(), v83.Note)
			}
		})
	}
}
