package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ResetMonsterAnimationWriter = "ResetMonsterAnimation"

// ResetMonsterAnimation is the clientbound RESET_MONSTER_ANIMATION packet
// (CMob::OnSuspendReset): the server tells the client to un-suspend a mob and
// reset its action layer.
//
// Byte layout (IDA-verified, identical across all 5 versions — a single Decode1):
//   - animate : bool — when true the client re-shows the layer, resets the action
//     layer and clears the suspended flag; when false the handler is a no-op.
//     (CInPacket::Decode1 gates the whole reset body.)
//
// IDA basis: CMob::OnSuspendReset — v83 @0x66c500 (`if (CInPacket::Decode1(a2))
// { … SetLayerZ; PrepareActionLayer; m_nSuspended=0; m_bDoFirstAttack=1; }`),
// v84 @0x682802, v87 @0x6a73cb, v95 @0x64acb0, jms @0x6e9c8d — every version reads
// exactly one Decode1 and no further wire bytes.
//
// packet-audit:fname CMob::OnSuspendReset
type ResetMonsterAnimation struct {
	animate bool
}

func NewResetMonsterAnimation(animate bool) ResetMonsterAnimation {
	return ResetMonsterAnimation{animate: animate}
}

func (m ResetMonsterAnimation) Animate() bool     { return m.animate }
func (m ResetMonsterAnimation) Operation() string { return ResetMonsterAnimationWriter }
func (m ResetMonsterAnimation) String() string {
	return fmt.Sprintf("animate [%t]", m.animate)
}

func (m ResetMonsterAnimation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.animate)
		return w.Bytes()
	}
}

func (m *ResetMonsterAnimation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.animate = r.ReadBool()
	}
}
