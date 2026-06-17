package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobRequestEscortInfoHandle = "MobRequestEscortInfo"

// MobRequestEscortInfo is the serverbound MOB_REQUEST_ESCORT_INFO packet
// (CMob::SendRequestEscortPath): the client asks the server for an escort mob's
// path info.
//
// Byte layout (IDA-verified, a single Encode4):
//   - mobCrc : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//
// IDA basis: CMob::SendRequestEscortPath — v95 @0x6411f0 (opcode 237), jms
// @0x6eff57 (opcode 0xCC/204): `if (IsActive) { ClearEscortInfo; COutPacket(op);
// Encode4(SecureFuse(mobId)) }`. v95/jms only — escort family absent in v83/v84/v87.
//
// packet-audit:fname CMob::SendRequestEscortPath
type MobRequestEscortInfo struct {
	mobCrc uint32
}

func (m MobRequestEscortInfo) MobCrc() uint32    { return m.mobCrc }
func (m MobRequestEscortInfo) Operation() string { return MobRequestEscortInfoHandle }
func (m MobRequestEscortInfo) String() string {
	return fmt.Sprintf("mobCrc [%d]", m.mobCrc)
}

func (m MobRequestEscortInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		return w.Bytes()
	}
}

func (m *MobRequestEscortInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
	}
}
