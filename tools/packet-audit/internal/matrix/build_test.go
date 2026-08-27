package matrix

import (
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

	t.Run("verified sibling escapes", func(t *testing.T) {
		in := newInputs()
		in.Reports["gms_v95"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
		in.Markers[EvKey{pkt, "gms_v95"}] = MarkerStatus{Found: true, Address: "0x6dbe50"}
		in.Evidence[EvKey{pkt, "gms_v95"}] = EvidenceStatus{Exists: true, Fresh: true, Address: "0x6dbe50"}

		m := Build(in, versionKeys)
		c := subCell(t, m, pkt, "gms_v95")
		if c.State != StateVerified || c.Note != "" || c.Opcode != -1 {
			t.Fatalf("gms_v95 cell = %v (%q, opcode=%d); want StateVerified, \"\", -1", c.State.Name(), c.Note, c.Opcode)
		}
	})

	t.Run("unverified sibling stays suppressed", func(t *testing.T) {
		in := newInputs()
		in.Reports["gms_v95"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
		// No marker, no evidence: gradeSubStructCell cannot reach StateVerified,
		// so the protectedWriters bypass must not fire.

		m := Build(in, versionKeys)
		c := subCell(t, m, pkt, "gms_v95")
		if c.State != StateIncomplete || c.Note != "no audit report" {
			t.Fatalf("gms_v95 cell = %v (%q); want StateIncomplete, \"no audit report\"", c.State.Name(), c.Note)
		}
	})

	t.Run("stale evidence stays suppressed", func(t *testing.T) {
		in := newInputs()
		in.Reports["gms_v95"] = map[string]LoadedReport{"NpcAskSlideMenuConversationDetail": report}
		in.Markers[EvKey{pkt, "gms_v95"}] = MarkerStatus{Found: true, Address: "0x6dbe50"}
		in.Evidence[EvKey{pkt, "gms_v95"}] = EvidenceStatus{Exists: true, Fresh: false}

		m := Build(in, versionKeys)
		c := subCell(t, m, pkt, "gms_v95")
		if c.State != StateIncomplete || c.Note != "no audit report" {
			t.Fatalf("gms_v95 cell = %v (%q); want StateIncomplete, \"no audit report\"", c.State.Name(), c.Note)
		}
	})
}
