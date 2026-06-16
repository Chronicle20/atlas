package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterSpecialEffectBySkillWriter = "MonsterSpecialEffectBySkill"

// MonsterSpecialEffectBySkill is the clientbound MONSTER_SPECIAL_EFFECT_BY_SKILL
// packet (CMob::OnSpecialEffectBySkill): the server tells the client to play a
// skill's "special" hit-effect UOL on a mob.
//
// Byte layout (IDA-verified — version-dependent):
//
//	v83/v84/v87/jms (single field):
//	  - skillId : int32 — the skill whose GetSpecialUOL is looked up client-side
//	v95 (three fields):
//	  - skillId   : int32  — skill id (Decode4)
//	  - characterId: int32 — the user the effect is attached to (Decode4 → GetUser)
//	  - delay     : uint16 — apply-start delay ms (Decode2 → tDelay)
//
// IDA basis: CMob::OnSpecialEffectBySkill —
//   - v83 @0x66d8e7: `v3 = Decode4(a2); GetSkill(...,v3); GetSpecialUOL(...)` — one
//     Decode4, the rest is computed from the skill entry (no further wire reads).
//   - v84 @0x683be9, v87 @0x6a87b3, jms @0x6eb08d: identical single-Decode4 shape.
//   - v95 @0x6540b0: `v4 = Decode4; GetSkill; v6 = Decode4; GetUser(...,v6);
//     v7 = Decode2; tDelay = v7` — three wire fields. The extra user-id + delay are
//     a GMS-95 addition (jms v185, though numerically > 95, keeps the single-field
//     shape), so the branch gates on GMS region AND major >= 95.
type MonsterSpecialEffectBySkill struct {
	skillId     int32
	characterId int32
	delay       uint16
}

func NewMonsterSpecialEffectBySkill(skillId int32, characterId int32, delay uint16) MonsterSpecialEffectBySkill {
	return MonsterSpecialEffectBySkill{skillId: skillId, characterId: characterId, delay: delay}
}

func (m MonsterSpecialEffectBySkill) SkillId() int32     { return m.skillId }
func (m MonsterSpecialEffectBySkill) CharacterId() int32 { return m.characterId }
func (m MonsterSpecialEffectBySkill) Delay() uint16      { return m.delay }
func (m MonsterSpecialEffectBySkill) Operation() string  { return MonsterSpecialEffectBySkillWriter }
func (m MonsterSpecialEffectBySkill) String() string {
	return fmt.Sprintf("skillId [%d], characterId [%d], delay [%d]", m.skillId, m.characterId, m.delay)
}

// v95Layout reports whether this tenant uses the three-field GMS-95 layout.
func v95SpecialEffectLayout(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(95)
}

func (m MonsterSpecialEffectBySkill) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.skillId)
		if v95SpecialEffectLayout(t) {
			w.WriteInt32(m.characterId)
			w.WriteShort(m.delay)
		}
		return w.Bytes()
	}
}

func (m *MonsterSpecialEffectBySkill) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.skillId = r.ReadInt32()
		if v95SpecialEffectLayout(t) {
			m.characterId = r.ReadInt32()
			m.delay = r.ReadUint16()
		}
	}
}
