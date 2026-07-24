package matrix

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
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
