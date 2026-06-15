package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobEscortStopEndRequestHandle = "MobEscortStopEndRequest"

// MobEscortStopEndRequest is the serverbound MOB_ESCORT_STOP_END_REQUEST packet
// (CMob::SendEscortStopEndRequest): the client asks the server to end an escort
// stop.
//
// Byte layout (IDA-verified, a single Encode4):
//   - mobCrc : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//
// IDA basis: CMob::SendEscortStopEndRequest — v95 @0x641290 (opcode 238), jms
// @0x6effcd (opcode 0xCD/205): `if (IsActive) { COutPacket(op);
// Encode4(SecureFuse(mobId)) }`. v95/jms only — escort family absent in v83/v84/v87.
//
// packet-audit:fname CMob::SendEscortStopEndRequest
type MobEscortStopEndRequest struct {
	mobCrc uint32
}

func (m MobEscortStopEndRequest) MobCrc() uint32    { return m.mobCrc }
func (m MobEscortStopEndRequest) Operation() string { return MobEscortStopEndRequestHandle }
func (m MobEscortStopEndRequest) String() string {
	return fmt.Sprintf("mobCrc [%d]", m.mobCrc)
}

func (m MobEscortStopEndRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		return w.Bytes()
	}
}

func (m *MobEscortStopEndRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
	}
}
