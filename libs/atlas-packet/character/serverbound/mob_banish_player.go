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
// IDA basis: CUserLocal::SendBanMapByMobRequest — v87 @0x9df571, v95 @0x908d50,
// jms @0xa28621 (COutPacket(opcode); Encode4(dwMobTemplateID); SendPacket).
// In v83/v84 this send is INLINED into CUserLocal::Update (no standalone function
// to pin — see structures/RESUME-STATE.md); the wire shape is byte-identical (the
// v87/v95 standalone is a one-Encode4 wrapper), so v83/v84 take this same codec.
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
