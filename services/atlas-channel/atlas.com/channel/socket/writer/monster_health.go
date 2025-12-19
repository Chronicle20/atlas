package writer

import (
	"atlas-channel/monster"
	"github.com/Chronicle20/atlas-socket/response"
	"math"
)

const MonsterHealth = "MonsterHealth"

func MonsterHealthBody(m monster.Model) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(m.UniqueId())
		rem := byte(math.Max(1, float64(m.HP())*100/float64(m.MaxHP())))
		w.WriteByte(rem)
		return w.Bytes()
	}
}
