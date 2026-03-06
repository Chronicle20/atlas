package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DestroyMonsterType byte

var DestroyMonsterTypeDisappear DestroyMonsterType = 0
var DestroyMonsterTypeFadeOut DestroyMonsterType = 1

const DestroyMonster = "DestroyMonster"

func DestroyMonsterBody(uniqueId uint32, destroyType DestroyMonsterType) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			w.WriteByte(byte(destroyType))
			return w.Bytes()
		}
	}
}
