package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const AutoAggroHandle = "AutoAggro"

// AutoAggro is the serverbound AUTO_AGGRO packet (CMob::ApplyControl): the
// client asks the server to make it the mob's aggro'd controller.
//
// CMob::Update calls ApplyControl unconditionally; it sends at most once per
// second, for any mob whose template carries bFirstAttack OR bPickUpDrop, from
// ANY client that can see the mob — controller or not. bPickUpDrop alone is
// enough to fire it, so the aggressive-template check is server-side
// (atlas-monsters SetAggro gate 3), never inferred from the packet's presence.
//
// Byte layout (IDA-verified, IDENTICAL across all ten versions — two Encode4;
// no version gate):
//   - mobId    : uint32 — CMob::m_dwMobID. The send site encodes
//     _ZtlSecureFuse(m_dwMobID, m_dwMobID_CS); fuse RECOVERS the plaintext
//     object id, so the wire carries the mob object id verbatim. The sibling
//     MobDropPickupRequest names the same value `mobCrc`; that name is a
//     misnomer inherited from the send site and is not propagated here.
//   - distance : uint32 — the client's own proximity score,
//     |dx|/10 + |dy|/3, +100 when nMoveAction & 0xFFFFFFFE == 0x12.
//     CMob::TryFirstAttack chases at score <= 40 (v95 @0x6482f0); the channel
//     admission gate adopts that bar.
//
// IDA basis: CMob::ApplyControl — v48 @0x551c79, v61 @0x5ccf1c, v72 @0x61d358,
// v79 @0x63d0e6, v83 @0x66e146, v84 @0x684492, v87 @0x6a9061, v92 @0x636320,
// v95 @0x640d20, jms_v185 @0x6eba3c.
//
// packet-audit:fname CMob::ApplyControl
type AutoAggro struct {
	mobId    uint32
	distance uint32
}

func NewAutoAggro(mobId uint32, distance uint32) AutoAggro {
	return AutoAggro{mobId: mobId, distance: distance}
}

func (m AutoAggro) MobId() uint32     { return m.mobId }
func (m AutoAggro) Distance() uint32  { return m.distance }
func (m AutoAggro) Operation() string { return AutoAggroHandle }
func (m AutoAggro) String() string {
	return fmt.Sprintf("mobId [%d], distance [%d]", m.mobId, m.distance)
}

func (m AutoAggro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobId)
		w.WriteInt(m.distance)
		return w.Bytes()
	}
}

func (m *AutoAggro) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobId = r.ReadUint32()
		m.distance = r.ReadUint32()
	}
}
