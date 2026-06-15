package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobEscortCollisionHandle = "MobEscortCollision"

// MobEscortCollision is the serverbound MOB_ESCORT_COLLISION packet
// (CMob::SendCollisionEscort): the client reports that an escort mob collided with
// (reached) an escort destination.
//
// Byte layout (IDA-verified, two Encode4):
//   - mobCrc : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//   - dest   : uint32 — the escort destination index (nDest)
//
// IDA basis: CMob::SendCollisionEscort — v95 @0x641150 (opcode 236), jms @0x6efeb7
// (opcode 0xCB/203): `COutPacket(op); Encode4(SecureFuse(mobId)); Encode4(nDest)`.
// v95/jms only — the escort family is absent in v83/v84/v87 (no SendCollisionEscort
// symbol, no escort dispatcher cases).
//
// packet-audit:fname CMob::SendCollisionEscort
type MobEscortCollision struct {
	mobCrc uint32
	dest   uint32
}

func (m MobEscortCollision) MobCrc() uint32    { return m.mobCrc }
func (m MobEscortCollision) Dest() uint32      { return m.dest }
func (m MobEscortCollision) Operation() string { return MobEscortCollisionHandle }
func (m MobEscortCollision) String() string {
	return fmt.Sprintf("mobCrc [%d], dest [%d]", m.mobCrc, m.dest)
}

func (m MobEscortCollision) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		w.WriteInt(m.dest)
		return w.Bytes()
	}
}

func (m *MobEscortCollision) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
		m.dest = r.ReadUint32()
	}
}
