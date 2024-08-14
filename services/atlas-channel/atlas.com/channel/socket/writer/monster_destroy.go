package writer

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DestroyMonsterType byte

var DestroyMonsterTypeDisappear DestroyMonsterType = 0
var DestroyMonsterTypeFadeOut DestroyMonsterType = 1

const DestroyMonster = "DestroyMonster"

func DestroyMonsterBody(_ logrus.FieldLogger, _ tenant.Model) func(uniqueId uint32, destroyType DestroyMonsterType) BodyProducer {
	return func(uniqueId uint32, destroyType DestroyMonsterType) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			w.WriteByte(byte(destroyType))
			return w.Bytes()
		}
	}
}
