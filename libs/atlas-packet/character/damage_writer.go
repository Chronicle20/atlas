package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterDamageWriter = "CharacterDamage"

type CharacterDamageW struct {
	characterId       uint32
	attackIdx         model.DamageType
	damage            int32
	monsterTemplateId uint32
	left              bool
}

func NewCharacterDamageW(characterId uint32, attackIdx model.DamageType, damage int32, monsterTemplateId uint32, left bool) CharacterDamageW {
	return CharacterDamageW{
		characterId:       characterId,
		attackIdx:         attackIdx,
		damage:            damage,
		monsterTemplateId: monsterTemplateId,
		left:              left,
	}
}

func (m CharacterDamageW) CharacterId() uint32       { return m.characterId }
func (m CharacterDamageW) AttackIdx() model.DamageType { return m.attackIdx }
func (m CharacterDamageW) DamageAmount() int32       { return m.damage }
func (m CharacterDamageW) MonsterTemplateId() uint32 { return m.monsterTemplateId }
func (m CharacterDamageW) Left() bool                { return m.left }
func (m CharacterDamageW) Operation() string         { return CharacterDamageWriter }
func (m CharacterDamageW) String() string {
	return fmt.Sprintf("characterId [%d], damage [%d]", m.characterId, m.damage)
}

func (m CharacterDamageW) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(byte(m.attackIdx))
		w.WriteInt32(m.damage)
		if m.attackIdx == model.DamageTypePhysical || m.attackIdx == model.DamageTypeMagic {
			w.WriteInt(m.monsterTemplateId)
			w.WriteBool(m.left)
			w.WriteBool(false) // stance
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteByte(0) // bGuard
			}
			w.WriteByte(0) // stance related
		}
		w.WriteInt32(m.damage)
		if m.damage == -1 {
			w.WriteInt(0) // misdirection skill
		}
		return w.Bytes()
	}
}

func (m *CharacterDamageW) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.attackIdx = model.DamageType(r.ReadInt8())
		m.damage = r.ReadInt32()
		if m.attackIdx == model.DamageTypePhysical || m.attackIdx == model.DamageTypeMagic {
			m.monsterTemplateId = r.ReadUint32()
			m.left = r.ReadBool()
			_ = r.ReadBool() // stance
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				_ = r.ReadByte() // bGuard
			}
			_ = r.ReadByte() // stance related
		}
		_ = r.ReadInt32() // damage repeated
		if m.damage == -1 {
			_ = r.ReadUint32() // misdirection skill
		}
	}
}
