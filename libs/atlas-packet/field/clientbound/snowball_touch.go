package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SnowballTouchWriter = "SnowballTouch"

// SnowballTouch models the LEFT_KNOCK_BACK clientbound packet
// (CField_SnowBall::OnSnowBallTouch).
//
// EMPTY payload — the packet carries the opcode only. The client handler reads
// no bytes from the CInPacket; it simply applies a knockback impulse via
// CUserLocal::SetImpact(0x12C, 1) (confirmed by disasm: push 1; push 12Ch;
// call SetImpact; retn — v84 @0x584ceb, v87 @0x5a35f7). All five versions
// share the empty body; only the opcode shifts.
// packet-audit:fname CField_SnowBall::OnSnowBallTouch
type SnowballTouch struct{}

func NewSnowballTouch() SnowballTouch {
	return SnowballTouch{}
}

func (m SnowballTouch) Operation() string { return SnowballTouchWriter }
func (m SnowballTouch) String() string {
	return "SnowballTouch"
}

func (m SnowballTouch) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *SnowballTouch) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
