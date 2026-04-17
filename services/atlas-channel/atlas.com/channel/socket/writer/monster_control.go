package writer

import (
	"atlas-channel/monster"
	"context"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type ControlMonsterType int8

var ControlMonsterTypeReset = ControlMonsterType(0)
var ControlMonsterTypeActiveInit = ControlMonsterType(1)
var ControlMonsterTypeActiveRequest = ControlMonsterType(2)
var ControlMonsterTypeActivePerm0 = ControlMonsterType(3)
var ControlMonsterTypeActivePerm1 = ControlMonsterType(4)
var ControlMonsterTypePassive = ControlMonsterType(-1)
var ControlMonsterTypePassive0 = ControlMonsterType(-2)
var ControlMonsterTypePassive1 = ControlMonsterType(-3)

func StartControlMonsterBody(m monster.Model, aggro bool) packet.Encode {
	if aggro {
		return ControlMonsterBody(m, ControlMonsterTypeActiveRequest)
	}
	return ControlMonsterBody(m, ControlMonsterTypeActiveInit)
}

func StopControlMonsterBody(m monster.Model) packet.Encode {
	return ControlMonsterBody(m, ControlMonsterTypeReset)
}

func ControlMonsterBody(m monster.Model, controlType ControlMonsterType) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			var mem packetmodel.MonsterModel
			if controlType > ControlMonsterTypeReset {
				mem = packetmodel.NewMonster(m.X(), m.Y(), m.Stance(), m.Fh(), packetmodel.MonsterAppearTypeRegen, m.Team())
				stat := buildMonsterTemporaryStat(l, t, m)
				mem.SetTemporaryStat(stat)
			}
			return monsterpkt.NewMonsterControl(monsterpkt.ControlType(controlType), m.UniqueId(), m.MonsterId(), mem).Encode(l, ctx)(options)
		}
	}
}
