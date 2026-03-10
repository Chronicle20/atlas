package writer

import (
	"context"

	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type DestroyMonsterType byte

var DestroyMonsterTypeDisappear DestroyMonsterType = 0
var DestroyMonsterTypeFadeOut DestroyMonsterType = 1

const DestroyMonster = "DestroyMonster"

func DestroyMonsterBody(uniqueId uint32, destroyType DestroyMonsterType) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return monsterpkt.NewMonsterDestroy(uniqueId, monsterpkt.DestroyType(destroyType)).Encode(l, ctx)
	}
}
