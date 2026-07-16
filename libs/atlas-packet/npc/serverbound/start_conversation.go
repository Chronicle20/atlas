package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NPCStartConversationHandle = "NPCStartConversationHandle"

// startConversationHasXY reports whether the NPC-click "talk to npc" packet
// carries the user's current x/y shorts after the npc oid.
//
// IDA-verified send sites: v61 sub_7B1403@0x7b1403 COutPacket(54) +
// Encode4(oid) + Encode2(x) + Encode2(y); v79 CUserLocal::TalkToNpc@0x8b7e10
// and v83+ likewise append x + y; JMS also includes them.
//
// v48 (GMS_v48_1_DEVM.exe port 13337) ALSO carries x + y: the field NPC-click
// / membership-NPC talk sender sub_568A2A@0x568a2a builds COutPacket(46) +
// Encode4(npcObjId @0x569297/0x569380) + Encode2(userX @0x5692b0/0x569399) +
// Encode2(userY @0x5692ca/0x5693b3). So v48 is oid+x+y, not oid-only.
// task-113 v48 Stage E.
//
// v72 (task-113 Phase 5, GMS_v72.1_U_DEVM.exe port 13339): the canonical v72
// NPC_TALK serverbound sender is CUserLocal::TalkToNpc @ 0x63FD91 (registry
// opcode 57, ida-discovered). Its three COutPacket(57) send-sites (0x63fe09,
// 0x64066f, 0x640857) each build Encode4(npcOid) + Encode2(userX) +
// Encode2(userY) — i.e. v72 ALSO carries x/y. The earlier "oid-only" fixture
// cited 0x70dd49, which in v72 is CUICharacterSaleDlg::OnCreate (a stale symbol
// copied from v48, where sub_70DD49 is the unrelated ItemCancel sender); the
// v72 IDA export note attributing an "Encode4(oid) ONLY" TalkToNpc to 0x70dd49
// was likewise mis-attributed. Verified false-pass — v72 now takes the
// x/y-present branch. Pre-v48 GMS (no IDB) stays oid-only.
func startConversationHasXY(t tenant.Model) bool {
	if !t.IsRegion("GMS") {
		return true // JMS and other regions carry x/y
	}
	return t.MajorAtLeast(72) || t.MajorVersion() == 61 || t.MajorVersion() == 48
}

// packet-audit:fname CUserLocal::TalkToNpc
type StartConversation struct {
	oid uint32
	x   int16
	y   int16
}

func (m StartConversation) Oid() uint32 {
	return m.oid
}

func (m StartConversation) X() int16 {
	return m.x
}

func (m StartConversation) Y() int16 {
	return m.y
}

func (m StartConversation) Operation() string {
	return NPCStartConversationHandle
}

func (m StartConversation) String() string {
	return fmt.Sprintf("oid [%d] x [%d] y [%d]", m.oid, m.x, m.y)
}

func (m StartConversation) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		t := tenant.MustFromContext(ctx)
		if startConversationHasXY(t) {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
		}
		return w.Bytes()
	}
}

func (m *StartConversation) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		m.oid = r.ReadUint32()
		if startConversationHasXY(t) {
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
		}
	}
}
