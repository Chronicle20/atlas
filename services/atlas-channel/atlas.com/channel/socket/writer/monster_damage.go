package writer

import (
	"atlas-channel/monster"

	"github.com/Chronicle20/atlas-socket/response"
)

type DamageType byte

// MonsterDamage CMob::OnDamaged
const (
	MonsterDamage = "MonsterDamage"

	DamageTypeUnk1 = DamageType(0)
	DamageTypeUnk2 = DamageType(1)
	DamageTypeUnk3 = DamageType(2)
)

func MonsterDamageBody(m monster.Model, damageType DamageType, damage uint32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(m.UniqueId())
		w.WriteByte(byte(damageType))
		w.WriteInt(damage)
		w.WriteInt(m.Hp())
		w.WriteInt(m.MaxHp())
		return w.Bytes()
	}
}
