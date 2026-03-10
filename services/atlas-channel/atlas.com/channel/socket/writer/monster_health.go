package writer

import (
	"atlas-channel/monster"
	"context"
	"math"

	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const MonsterHealth = "MonsterHealth"

func MonsterHealthBody(m monster.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		hpPercent := byte(math.Max(1, float64(m.Hp())*100/float64(m.MaxHp())))
		return monsterpkt.NewMonsterHealth(m.UniqueId(), hpPercent).Encode(l, ctx)
	}
}
