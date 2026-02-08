package writer

import (
	"atlas-channel/monster"
	"math"

	"github.com/Chronicle20/atlas-socket/response"
)

const MonsterHealth = "MonsterHealth"

func MonsterHealthBody(m monster.Model) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(m.UniqueId())
		rem := byte(math.Max(1, float64(m.Hp())*100/float64(m.MaxHp())))
		w.WriteByte(rem)
		return w.Bytes()
	}
}
