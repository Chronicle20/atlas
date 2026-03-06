package writer

import (
	"atlas-channel/monster"
	"context"
	"math"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterHealth = "MonsterHealth"

func MonsterHealthBody(m monster.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(m.UniqueId())
			rem := byte(math.Max(1, float64(m.Hp())*100/float64(m.MaxHp())))
			w.WriteByte(rem)
			return w.Bytes()
		}
	}
}
