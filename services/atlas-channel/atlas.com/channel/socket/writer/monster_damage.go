package writer

import (
	"atlas-channel/monster"
	"context"

	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
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
		return monsterpkt.NewMonsterDamage(m.UniqueId(), monsterpkt.MonsterDamageType(damageType), damage, m.Hp(), m.MaxHp()).Encode(l, ctx)
	}
}
