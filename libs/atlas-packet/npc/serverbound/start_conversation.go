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
		// GMS v72 TalkToNpc (sub_70DD49@0x70dd49, GMS_v72.1_U_DEVM.exe port 13339;
		// the sole COutPacket(57) sender, called from the CUserLocal NPC-click
		// path sub_69FE41 as sub_70DD49(-npcOid)) encodes ONLY Encode4(oid) — the
		// user-position x/y shorts were added after the legacy range (v79
		// CUserLocal::TalkToNpc@0x8b7e10 appends Encode2 x + Encode2 y). Legacy
		// GMS (<79) omits both. delta §3.2
		t := tenant.MustFromContext(ctx)
		if !(t.IsRegion("GMS") && !t.MajorAtLeast(79)) {
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
		if !(t.IsRegion("GMS") && !t.MajorAtLeast(79)) {
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
		}
	}
}
