package writer

import (
	"atlas-channel/monster"
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SpawnMonster = "SpawnMonster"

func SpawnMonsterBody(l logrus.FieldLogger, t tenant.Model) func(m monster.Model, newSpawn bool) BodyProducer {
	return func(m monster.Model, newSpawn bool) BodyProducer {
		return SpawnMonsterWithEffectBody(l, t)(m, newSpawn, 0)
	}
}

func SpawnMonsterWithEffectBody(l logrus.FieldLogger, t tenant.Model) func(m monster.Model, newSpawn bool, effect byte) BodyProducer {
	return func(m monster.Model, newSpawn bool, effect byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(m.UniqueId())
			if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
				if m.Controlled() {
					w.WriteByte(1)
				} else {
					w.WriteByte(5)
				}
			}
			w.WriteInt(m.MonsterId())

			appearType := model.MonsterAppearTypeNormal
			if newSpawn {
				appearType = model.MonsterAppearTypeRegen
			}

			mem := model.NewMonster(m.X(), m.Y(), m.Stance(), m.Fh(), appearType, m.Team())
			mem.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}
