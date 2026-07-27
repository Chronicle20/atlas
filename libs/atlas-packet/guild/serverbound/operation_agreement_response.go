package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// AgreementResponse is the serverbound member reply to a guild-creation agree
// dialog. CField::SendCreateGuildAgreeMsg writes Encode1(op) + Encode4(unk) +
// Encode1(agreed); after the GUILD_OPERATION dispatcher strips the leading op
// byte the body is Encode4(unk) + Encode1(agreed) — i.e. the `unk` 4-byte int
// (the party/guild id from CWvsContext, *m_pStr[..]) IS a real wire field, NOT
// an extra to drop. IDA-verified byte-correct across every version (task-103):
//
//	v83 @0x530666 : Encode1(0x1E) + Encode4(*szCookie[92])  + Encode1(a2)
//	v87 @0x557e6e : Encode1(0x1E) + Encode4(*szCookie[100]) + Encode1(a2)
//	v95 @0x52d780 : Encode1(0x20) + Encode4(*m_pStr[2093])  + Encode1(bAgree)
//	jms @0x56da47 : Encode1(0x1E) + Encode4(*szCookie[88])  + Encode1(bAgree)
//
// (The prior run.go "❌ wire mismatch — extra Encode4 unk" note was STALE/wrong;
// the existing codec already matches the client. No wire change made.)
// packet-audit:fname CField::SendCreateGuildAgreeMsg
type AgreementResponse struct {
	unk    uint32
	agreed bool
}

func (m AgreementResponse) Unk() uint32  { return m.unk }
func (m AgreementResponse) Agreed() bool { return m.agreed }

func (m AgreementResponse) Operation() string { return "AgreementResponse" }

func (m AgreementResponse) String() string {
	return fmt.Sprintf("unk [%d] agreed [%t]", m.unk, m.agreed)
}

func (m AgreementResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.unk)
		w.WriteBool(m.agreed)
		return w.Bytes()
	}
}

func (m *AgreementResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.unk = r.ReadUint32()
		m.agreed = r.ReadBool()
	}
}
