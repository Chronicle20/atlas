package writer

import (
	"atlas-channel/monster"
	"atlas-channel/socket/model"
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

func StartControlMonsterBody(l logrus.FieldLogger, t tenant.Model) func(m monster.Model, aggro bool) BodyProducer {
	return func(m monster.Model, aggro bool) BodyProducer {
		if aggro {
			return ControlMonsterBody(l, t)(m, ControlMonsterTypeActiveRequest)
		}
		return ControlMonsterBody(l, t)(m, ControlMonsterTypeActiveInit)
	}
}

func StopControlMonsterBody(l logrus.FieldLogger, t tenant.Model) func(m monster.Model) BodyProducer {
	return func(m monster.Model) BodyProducer {
		return ControlMonsterBody(l, t)(m, ControlMonsterTypeReset)
	}
}

func ControlMonsterBody(l logrus.FieldLogger, t tenant.Model) func(m monster.Model, controlType ControlMonsterType) BodyProducer {
	return func(m monster.Model, controlType ControlMonsterType) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt8(int8(controlType))
			w.WriteInt(m.UniqueId())
			if controlType > ControlMonsterTypeReset {
				w.WriteByte(5)
				w.WriteInt(m.MonsterId())
				mem := model.NewMonster(m.X(), m.Y(), m.Stance(), m.Fh(), model.MonsterAppearTypeRegen, m.Team())
				mem.Encode(l, t, options)(w)
				return w.Bytes()
			}
			return w.Bytes()
		}
	}
}
