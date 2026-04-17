package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterDamageWriter = "MonsterDamage"

type MonsterDamageType byte

const (
	MonsterDamageTypeUnk1 MonsterDamageType = 0
	MonsterDamageTypeUnk2 MonsterDamageType = 1
	MonsterDamageTypeUnk3 MonsterDamageType = 2
)

type Damage struct {
	uniqueId   uint32
	damageType MonsterDamageType
	damage     uint32
	hp         uint32
	maxHp      uint32
}

func NewMonsterDamage(uniqueId uint32, damageType MonsterDamageType, damage uint32, hp uint32, maxHp uint32) Damage {
	return Damage{uniqueId: uniqueId, damageType: damageType, damage: damage, hp: hp, maxHp: maxHp}
}

func (m Damage) UniqueId() uint32          { return m.uniqueId }
func (m Damage) DamageType() MonsterDamageType { return m.damageType }
func (m Damage) DamageAmount() uint32      { return m.damage }
func (m Damage) Hp() uint32               { return m.hp }
func (m Damage) MaxHp() uint32             { return m.maxHp }
func (m Damage) Operation() string         { return MonsterDamageWriter }
func (m Damage) String() string {
	return fmt.Sprintf("uniqueId [%d], damage [%d], hp [%d/%d]", m.uniqueId, m.damage, m.hp, m.maxHp)
}

func (m Damage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(byte(m.damageType))
		w.WriteInt(m.damage)
		w.WriteInt(m.hp)
		w.WriteInt(m.maxHp)
		return w.Bytes()
	}
}

func (m *Damage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.damageType = MonsterDamageType(r.ReadByte())
		m.damage = r.ReadUint32()
		m.hp = r.ReadUint32()
		m.maxHp = r.ReadUint32()
	}
}
