package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobBanishPlayerHandle = "MobBanishPlayer"

// MobBanishPlayer is the serverbound MOB_BANISH_PLAYER packet
// (CUserLocal::SendBanMapByMobRequest): the client requests that a mob banish the
// player from the map (e.g. boss "banish on touch" mechanics), citing the mob
// template that triggered it.
//
// Byte layout (IDA-verified): a single Encode4(dwMobTemplateID).
//   - mobTemplateId : uint32 — the template id of the banishing mob
//
// IDA basis: CUserLocal::SendBanMapByMobRequest is a discrete one-Encode4 wrapper
// in ALL five clients — v83 @0x99b16a, v84 @0x99b173, v87 @0x9df571, v95 @0x908d50,
// jms @0xa28621 (COutPacket(opcode); Encode4(dwMobTemplateID); SendPacket). (task-092
// Stage 4 corrected the earlier "v83/v84 inlined" note — those were just unnamed
// sub_XXXX functions, now named + pinned.) Wire shape is byte-identical, one codec.
type MobBanishPlayer struct {
	mobTemplateId uint32
}

func (m MobBanishPlayer) MobTemplateId() uint32 { return m.mobTemplateId }
func (m MobBanishPlayer) Operation() string     { return MobBanishPlayerHandle }
func (m MobBanishPlayer) String() string {
	return fmt.Sprintf("mobTemplateId [%d]", m.mobTemplateId)
}

func (m MobBanishPlayer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobTemplateId)
		return w.Bytes()
	}
}

func (m *MobBanishPlayer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobTemplateId = r.ReadUint32()
	}
}
