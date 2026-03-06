package writer

import (
	"atlas-channel/monster"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
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

const ControlMonster = "ControlMonster"

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
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt8(int8(controlType))
			w.WriteInt(m.UniqueId())
			if controlType > ControlMonsterTypeReset {
				w.WriteByte(5)
				w.WriteInt(m.MonsterId())
				mem := model.NewMonster(m.X(), m.Y(), m.Stance(), m.Fh(), model.MonsterAppearTypeRegen, m.Team())
				stat := buildMonsterTemporaryStat(l, t, m)
				mem.SetTemporaryStat(stat)
				w.WriteByteArray(mem.Encoder(l, ctx)(options))
				return w.Bytes()
			}
			return w.Bytes()
		}
	}
}
