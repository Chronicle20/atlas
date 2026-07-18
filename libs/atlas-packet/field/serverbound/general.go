package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CharacterChatGeneralHandle = "CharacterChatGeneralHandle"

// packet-audit:fname CField::SendChatMsg
type General struct {
	updateTime   uint32
	msg          string
	bOnlyBalloon bool
}

func (m General) UpdateTime() uint32 {
	return m.updateTime
}

func (m General) Msg() string {
	return m.msg
}

func (m General) BOnlyBalloon() bool {
	return m.bOnlyBalloon
}

func (m General) Operation() string {
	return CharacterChatGeneralHandle
}

func (m General) String() string {
	return fmt.Sprintf("msg [%s] updateTime [%d] bOnlyBalloon [%t]", m.msg, m.updateTime, m.bOnlyBalloon)
}

func (m General) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
			// updateTime is a later GMS chat field; v84..86 == v83 (off-by-one fix).
			// delta §3.1.10. MED: 87-vs-95 unpinned in A4; >=87 excludes v84, keeps v83/v95.
			w.WriteInt(m.updateTime)
		}
		w.WriteAsciiString(m.msg)
		// bOnlyBalloon is absent on the oldest GMS client (v48). IDA: the CField
		// chat send helper sub_4C3DEF @0x4c3def builds COutPacket(40) @0x4c3e48 +
		// EncodeStr(msg) @0x4c3e67 + SendPacket @0x4c3e76 with NO trailing byte.
		// v61's parser (sub_4E7469) adds Encode1(bOnlyBalloon); the balloon flag is
		// a >=61 addition. Gate excludes GMS<61 (v48) only; v61+/JMS unchanged.
		if !(t.IsRegion("GMS") && t.MajorVersion() < 61) {
			w.WriteBool(m.bOnlyBalloon)
		}
		return w.Bytes()
	}
}

func (m *General) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
			// updateTime is a later GMS chat field; v84..86 == v83 (off-by-one fix). delta §3.1.10
			m.updateTime = r.ReadUint32()
		}
		m.msg = r.ReadAsciiString()
		// bOnlyBalloon absent on GMS<61 (v48): see Encode comment (sub_4C3DEF).
		if !(t.IsRegion("GMS") && t.MajorVersion() < 61) {
			m.bOnlyBalloon = r.ReadBool()
		}
	}
}
