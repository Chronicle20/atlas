package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	CharacterAttackMeleeWriter  = "CharacterAttackMelee"
	CharacterAttackRangedWriter = "CharacterAttackRanged"
	CharacterAttackMagicWriter  = "CharacterAttackMagic"
	CharacterAttackEnergyWriter = "CharacterAttackEnergy"
)

// Attack is a common attack packet for all 4 attack types.
// Service layer pre-computes mastery, bulletItemId, and skillLevel.
type Attack struct {
	attackType      string
	characterId     uint32
	level           byte
	skillLevel      byte
	skillId         uint32
	isStrafe        bool
	isMesoExplosion bool
	hasKeydown      bool
	mastery         byte
	bulletItemId    uint32
	attackInfo      model.AttackInfo
}

func NewAttackMelee(characterId uint32, level byte, skillLevel byte, mastery byte, bulletItemId uint32, isMesoExplosion bool, hasKeydown bool, ai model.AttackInfo) Attack {
	return newAttack(CharacterAttackMeleeWriter, characterId, level, skillLevel, mastery, bulletItemId, false, isMesoExplosion, hasKeydown, ai)
}

func NewAttackRanged(characterId uint32, level byte, skillLevel byte, mastery byte, bulletItemId uint32, isStrafe bool, hasKeydown bool, ai model.AttackInfo) Attack {
	return newAttack(CharacterAttackRangedWriter, characterId, level, skillLevel, mastery, bulletItemId, isStrafe, false, hasKeydown, ai)
}

func NewAttackMagic(characterId uint32, level byte, skillLevel byte, mastery byte, bulletItemId uint32, hasKeydown bool, ai model.AttackInfo) Attack {
	return newAttack(CharacterAttackMagicWriter, characterId, level, skillLevel, mastery, bulletItemId, false, false, hasKeydown, ai)
}

func NewAttackEnergy(characterId uint32, level byte, skillLevel byte, mastery byte, bulletItemId uint32, hasKeydown bool, ai model.AttackInfo) Attack {
	return newAttack(CharacterAttackEnergyWriter, characterId, level, skillLevel, mastery, bulletItemId, false, false, hasKeydown, ai)
}

func newAttack(attackType string, characterId uint32, level byte, skillLevel byte, mastery byte, bulletItemId uint32, isStrafe bool, isMesoExplosion bool, hasKeydown bool, ai model.AttackInfo) Attack {
	return Attack{
		attackType:      attackType,
		characterId:     characterId,
		level:           level,
		skillLevel:      skillLevel,
		skillId:         ai.SkillId(),
		isStrafe:        isStrafe,
		isMesoExplosion: isMesoExplosion,
		hasKeydown:      hasKeydown,
		mastery:         mastery,
		bulletItemId:    bulletItemId,
		attackInfo:      ai,
	}
}

func (m Attack) Operation() string { return m.attackType }
func (m Attack) String() string {
	return fmt.Sprintf("attack type [%s] characterId [%d] skillId [%d]", m.attackType, m.characterId, m.skillId)
}

func (m Attack) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		ai := m.attackInfo
		w.WriteInt(m.characterId)
		w.WriteByte(byte(ai.Damage()<<4 | uint32(ai.Hits())))
		w.WriteByte(m.level)
		if ai.SkillId() > 0 {
			w.WriteByte(m.skillLevel)
			w.WriteInt(ai.SkillId())
		} else {
			w.WriteByte(0)
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.isStrafe {
				w.WriteByte(0) // passive SLV
			}
		}
		w.WriteByte(ai.Option())
		left := 0
		if ai.Left() {
			left = 1
		}
		w.WriteInt16(int16((left << 15) | ai.AttackAction()))
		if ai.AttackAction() <= 0x110 {
			w.WriteByte(ai.ActionSpeed())
			w.WriteByte(m.mastery)
			w.WriteInt(m.bulletItemId)

			for _, di := range ai.DamageInfo() {
				w.WriteInt(di.MonsterId())
				if di.MonsterId() > 0 {
					w.WriteByte(di.HitAction())
					if m.isMesoExplosion {
						w.WriteByte(byte(len(di.Damages())))
					}
					for _, d := range di.Damages() {
						w.WriteInt(d)
					}
				}
			}
		}

		if ai.AttackType() == model.AttackTypeRanged {
			w.WriteShort(ai.BulletX())
			w.WriteShort(ai.BulletY())
		}
		if m.hasKeydown {
			w.WriteInt(ai.Keydown())
		}
		return w.Bytes()
	}
}

func (m *Attack) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: attack display packets are server-send-only with complex
		// conditional encoding (variable damage counts, skill-dependent fields).
	}
}
