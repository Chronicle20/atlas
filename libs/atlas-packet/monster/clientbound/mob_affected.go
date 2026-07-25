package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobAffectedWriter = "MobAffected"

// MobAffected is the clientbound MOB_AFFECTED packet (CMob::OnAffected): the
// server informs the client that a mob is under a skill's area-affect, so the
// client can draw the affected-skill overlay for a bounded duration.
//
// Byte layout (IDA-verified, identical across all 5 versions — Decode4 + Decode2):
//   - skillId : int32  — the affecting skill id (entry.nSkillID = Decode4)
//   - delay   : uint16 — ms until the overlay starts; the client sets
//     tStart = delay + get_update_time() (Decode2)
//
// IDA basis: CMob::OnAffected — v83 @0x66c675 (`v4 = Decode4(a2); v5 = Decode2(a2);
// entry->nSkillID = v4; entry->tStart = v5 + get_update_time()`), v84 @0x682977,
// v87 @0x6a7540, v95 @0x644400, jms @0x6e9df6 — every version reads exactly one
// Decode4 then one Decode2.
//
// packet-audit:fname CMob::OnAffected
type MobAffected struct {
	skillId int32
	delay   uint16
}

func NewMobAffected(skillId int32, delay uint16) MobAffected {
	return MobAffected{skillId: skillId, delay: delay}
}

func (m MobAffected) SkillId() int32    { return m.skillId }
func (m MobAffected) Delay() uint16     { return m.delay }
func (m MobAffected) Operation() string { return MobAffectedWriter }
func (m MobAffected) String() string {
	return fmt.Sprintf("skillId [%d], delay [%d]", m.skillId, m.delay)
}

func (m MobAffected) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.skillId)
		w.WriteShort(m.delay)
		return w.Bytes()
	}
}

func (m *MobAffected) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.skillId = r.ReadInt32()
		m.delay = r.ReadUint16()
	}
}
