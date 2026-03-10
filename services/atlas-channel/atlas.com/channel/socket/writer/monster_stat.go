package writer

import (
	"atlas-channel/socket/model"
	"context"

	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const MonsterStatSet = "MonsterStatSet"
const MonsterStatReset = "MonsterStatReset"

func MonsterStatSetBody(uniqueId uint32, stat *model.MonsterTemporaryStat) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return monsterpkt.NewMonsterStatSet(uniqueId, stat).Encode(l, ctx)
	}
}

func MonsterStatResetBody(uniqueId uint32, stat *model.MonsterTemporaryStat) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return monsterpkt.NewMonsterStatReset(uniqueId, stat).Encode(l, ctx)
	}
}
