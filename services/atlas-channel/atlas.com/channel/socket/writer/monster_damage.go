package writer

import (
	"atlas-channel/monster"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DamageType byte

// MonsterDamage CMob::OnDamaged
const (
	MonsterDamage = "MonsterDamage"

	DamageTypeUnk1 = DamageType(0)
	DamageTypeUnk2 = DamageType(1)
	DamageTypeUnk3 = DamageType(2)
)

func MonsterDamageBody(m monster.Model, damageType DamageType, damage uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(m.UniqueId())
			w.WriteByte(byte(damageType))
			w.WriteInt(damage)
			w.WriteInt(m.Hp())
			w.WriteInt(m.MaxHp())
			return w.Bytes()
		}
	}
}
