package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// NewAttackForDecode creates an Attack with the constructor flags needed to drive
// non-self-describing Decode branches. Data fields are populated by Decode.
func NewAttackForDecode(attackType string, skillId uint32, isStrafe bool, isMesoExplosion bool, hasKeydown bool) Attack {
	return Attack{
		attackType:      attackType,
		skillId:         skillId,
		isStrafe:        isStrafe,
		isMesoExplosion: isMesoExplosion,
		hasKeydown:      hasKeydown,
	}
}

func (m Attack) Operation() string    { return m.attackType }
func (m Attack) CharacterId() uint32   { return m.characterId }
func (m Attack) Level() byte           { return m.level }
func (m Attack) SkillLevel() byte      { return m.skillLevel }
func (m Attack) SkillId() uint32       { return m.skillId }
func (m Attack) Mastery() byte         { return m.mastery }
func (m Attack) BulletItemId() uint32  { return m.bulletItemId }
func (m Attack) AttackInfo() model.AttackInfo { return m.attackInfo }
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

func (m *Attack) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		packed := r.ReadByte()
		damage := uint32((packed >> 4) & 0x0F)
		hits := packed & 0x0F

		m.level = r.ReadByte()

		if m.skillId > 0 {
			m.skillLevel = r.ReadByte()
			m.skillId = r.ReadUint32()
		} else {
			_ = r.ReadByte() // zero byte
		}

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.isStrafe {
				_ = r.ReadByte() // passive SLV
			}
		}

		option := r.ReadByte()
		mask := r.ReadUint16()
		left := (mask >> 15) & 1
		attackAction := int(mask & 0x7FFF)

		var at model.AttackType
		switch m.attackType {
		case CharacterAttackMeleeWriter:
			at = model.AttackTypeMelee
		case CharacterAttackRangedWriter:
			at = model.AttackTypeRanged
		case CharacterAttackMagicWriter:
			at = model.AttackTypeMagic
		case CharacterAttackEnergyWriter:
			at = model.AttackTypeEnergy
		}

		ai := model.NewAttackInfo(at)
		ai.SetDamage(damage)
		ai.SetHits(hits)
		ai.SetSkillId(m.skillId)
		ai.SetOption(option)
		ai.SetLeft(left == 1)
		ai.SetAttackAction(attackAction)

		if attackAction <= 0x110 {
			ai.SetActionSpeed(r.ReadByte())
			m.mastery = r.ReadByte()
			m.bulletItemId = r.ReadUint32()

			for range damage {
				monsterId := r.ReadUint32()
				di := model.NewDamageInfo(hits)
				di.SetMonsterId(monsterId)
				if monsterId > 0 {
					di.SetHitAction(r.ReadByte())
					damageCount := hits
					if m.isMesoExplosion {
						damageCount = r.ReadByte()
					}
					damages := make([]uint32, damageCount)
					for j := range damageCount {
						damages[j] = r.ReadUint32()
					}
					di.SetDamages(damages)
				}
				ai.AddDamageInfo(*di)
			}
		}

		if at == model.AttackTypeRanged {
			ai.SetBulletPosition(r.ReadUint16(), r.ReadUint16())
		}
		if m.hasKeydown {
			ai.SetKeydown(r.ReadUint32())
		}

		m.attackInfo = *ai
	}
}
