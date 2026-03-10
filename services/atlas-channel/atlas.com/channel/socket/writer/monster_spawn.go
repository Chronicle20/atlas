package writer

import (
	"atlas-channel/monster"
	"atlas-channel/socket/model"
	"context"

	monsterpkt "github.com/Chronicle20/atlas-packet/monster"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SpawnMonster = "SpawnMonster"

func buildMonsterTemporaryStat(l logrus.FieldLogger, t tenant.Model, m monster.Model) *model.MonsterTemporaryStat {
	stat := model.NewMonsterTemporaryStat()
	for _, se := range m.StatusEffects() {
		for name, value := range se.Statuses() {
			stat.AddStat(l)(t)(name, se.SourceSkillId(), se.SourceSkillLevel(), value, se.ExpiresAt())
		}
	}
	return stat
}

func SpawnMonsterBody(m monster.Model, newSpawn bool) packet.Encode {
	return SpawnMonsterWithEffectBody(m, newSpawn, 0)
}

func SpawnMonsterWithEffectBody(m monster.Model, newSpawn bool, effect byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			appearType := model.MonsterAppearTypeNormal
			if newSpawn {
				appearType = model.MonsterAppearTypeRegen
			}

			mem := model.NewMonster(m.X(), m.Y(), m.Stance(), m.Fh(), appearType, m.Team())
			stat := buildMonsterTemporaryStat(l, t, m)
			mem.SetTemporaryStat(stat)

			return monsterpkt.NewMonsterSpawn(m.UniqueId(), m.Controlled(), m.MonsterId(), mem).Encode(l, ctx)(options)
		}
	}
}
