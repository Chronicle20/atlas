package writer

import (
	dmap "atlas-channel/data/map"
	"atlas-channel/monster"
	"context"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func buildMonsterTemporaryStat(l logrus.FieldLogger, t tenant.Model, m monster.Model) *packetmodel.MonsterTemporaryStat {
	stat := packetmodel.NewMonsterTemporaryStat()
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
			appearType := packetmodel.MonsterAppearTypeNormal
			if newSpawn {
				appearType = packetmodel.MonsterAppearTypeRegen
			}

			// Snap mob position to (foothold surface - 1) before encoding so
			// the v83 client's spawn-packet validation doesn't drop the mob
			// through the floor. See data/map.SnapMobPosition.
			x, y := dmap.SnapMobPosition(l, ctx, m.MapId(), m.X(), m.Y(), m.Fh())

			// Debug: capture exactly what the wire spawn packet carries.
			// Useful when investigating fall-through reports — lets us
			// correlate the (x, y, fh) the client received against what the
			// client subsequently does with the mob (drop, walk, etc.).
			l.Debugf("Spawn monster wire: uniqueId=[%d] monsterId=[%d] x=[%d] y=[%d] fh=[%d] stance=[%d] newSpawn=[%t] controlled=[%t]",
				m.UniqueId(), m.MonsterId(), x, y, m.Fh(), m.Stance(), newSpawn, m.Controlled())

			mem := packetmodel.NewMonster(x, y, m.Stance(), m.Fh(), appearType, m.Team())
			stat := buildMonsterTemporaryStat(l, t, m)
			mem.SetTemporaryStat(stat)

			return monsterpkt.NewMonsterSpawn(m.UniqueId(), m.Controlled(), m.MonsterId(), mem).Encode(l, ctx)(options)
		}
	}
}
