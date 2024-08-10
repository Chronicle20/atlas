package writer

import (
	"atlas-channel/monster"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DestroyMonsterType byte

var DestroyMonsterTypeDissapear DestroyMonsterType = 0
var DestroyMonsterTypeFadeOut DestroyMonsterType = 1

const DestroyMonster = "DestroyMonster"

func DestroyMonsterBody(_ logrus.FieldLogger, _ tenant.Model) func(m monster.Model, destroyType DestroyMonsterType) BodyProducer {
	return func(m monster.Model, destroyType DestroyMonsterType) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(m.UniqueId())
			w.WriteByte(byte(destroyType))
			return w.Bytes()
		}
	}
}
