package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobDropPickupRequestHandle = "MobDropPickupRequest"

// MobDropPickupRequest is the serverbound MOB_DROP_PICKUP_REQUEST packet
// (CMob::SendDropPickUpRequest): the client asks the server to let a mob pick up
// a nearby drop.
//
// Byte layout (IDA-verified, identical across all 5 versions — two Encode4):
//   - mobCrc : uint32 — secured mob id (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS))
//   - dropId : uint32 — the drop object id the mob is trying to pick up
//
// IDA basis: CMob::SendDropPickUpRequest — v83 @0x66e91f, v87 @0x6a98ae,
// v95 @0x644450:
//   COutPacket(opcode); Encode4(_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS)); Encode4(dwDropID)
//
// packet-audit:fname CMob::SendDropPickUpRequest
type MobDropPickupRequest struct {
	mobCrc uint32
	dropId uint32
}

func (m MobDropPickupRequest) MobCrc() uint32    { return m.mobCrc }
func (m MobDropPickupRequest) DropId() uint32    { return m.dropId }
func (m MobDropPickupRequest) Operation() string { return MobDropPickupRequestHandle }
func (m MobDropPickupRequest) String() string {
	return fmt.Sprintf("mobCrc [%d], dropId [%d]", m.mobCrc, m.dropId)
}

func (m MobDropPickupRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobCrc)
		w.WriteInt(m.dropId)
		return w.Bytes()
	}
}

func (m *MobDropPickupRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobCrc = r.ReadUint32()
		m.dropId = r.ReadUint32()
	}
}
