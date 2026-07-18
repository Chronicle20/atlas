package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobAttackedByMobWriter = "MobAttackedByMob"

// MobAttackedByMob is the clientbound MOB_ATTACKED_BY_MOB packet
// (CMob::OnMobAttackedByMob): the server tells the client a mob took damage from
// another mob (mob-vs-mob, e.g. an escort attacker); the client shows the damage
// number and, when the attack index resolves, plays the attacker's hit effect.
//
// Byte layout (IDA-verified):
//   - attackIndex   : int8  — Decode1, signed; the attacker's attack-info index
//   - damage        : int32 — Decode4, damage to show (CMob::ShowDamage)
//   - mobTemplateId : int32 — Decode4, attacker mob template id   } only present
//   - left          : bool  — Decode1, attacker facing            } when attackIndex > -2
//
// The trailing two fields sit behind the client guard `if (attackIndex > -2)`. The
// server always emits a real attack (attackIndex >= 0), so the full 4-field form is
// the on-wire shape; the codec models all four and documents the guard.
//
// IDA basis: CMob::OnMobAttackedByMob — v83 @0x670f41, v84 @0x68749a,
// v87 @0x6ac074, v95 @0x6436a0, jms @0x6ee151. Every version: Decode1 attackIndex,
// Decode4 damage, then under `>-2`: Decode4 mobTemplateId, Decode1 left.
//
// packet-audit:fname CMob::OnMobAttackedByMob
type MobAttackedByMob struct {
	attackIndex   int8
	damage        int32
	mobTemplateId int32
	left          bool
}

func NewMobAttackedByMob(attackIndex int8, damage int32, mobTemplateId int32, left bool) MobAttackedByMob {
	return MobAttackedByMob{attackIndex: attackIndex, damage: damage, mobTemplateId: mobTemplateId, left: left}
}

func (m MobAttackedByMob) AttackIndex() int8    { return m.attackIndex }
func (m MobAttackedByMob) Damage() int32        { return m.damage }
func (m MobAttackedByMob) MobTemplateId() int32 { return m.mobTemplateId }
func (m MobAttackedByMob) Left() bool           { return m.left }
func (m MobAttackedByMob) Operation() string    { return MobAttackedByMobWriter }
func (m MobAttackedByMob) String() string {
	return fmt.Sprintf("attackIndex [%d], damage [%d], mobTemplateId [%d], left [%t]", m.attackIndex, m.damage, m.mobTemplateId, m.left)
}

func (m MobAttackedByMob) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt8(m.attackIndex)
		w.WriteInt32(m.damage)
		// guard: client reads these only when attackIndex > -2 (Decode4 + Decode1)
		if m.attackIndex > -2 {
			w.WriteInt32(m.mobTemplateId)
			w.WriteBool(m.left)
		}
		return w.Bytes()
	}
}

func (m *MobAttackedByMob) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.attackIndex = r.ReadInt8()
		m.damage = r.ReadInt32()
		if m.attackIndex > -2 {
			m.mobTemplateId = r.ReadInt32()
			m.left = r.ReadBool()
		}
	}
}
