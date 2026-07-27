package writer

import (
	dmap "atlas-channel/data/map"
	"atlas-channel/monster"
	"context"

	"github.com/sirupsen/logrus"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type ControlMonsterType int8

var (
	ControlMonsterTypeReset         = ControlMonsterType(0)
	ControlMonsterTypeActiveInit    = ControlMonsterType(1)
	ControlMonsterTypeActiveRequest = ControlMonsterType(2)
	ControlMonsterTypeActivePerm0   = ControlMonsterType(3)
	ControlMonsterTypeActivePerm1   = ControlMonsterType(4)
	ControlMonsterTypePassive       = ControlMonsterType(-1)
	ControlMonsterTypePassive0      = ControlMonsterType(-2)
	ControlMonsterTypePassive1      = ControlMonsterType(-3)
)

func StartControlMonsterBody(m monster.Model, aggro bool) packet.Encode {
	if aggro {
		return ControlMonsterBody(m, ControlMonsterTypeActiveRequest, true)
	}
	return ControlMonsterBody(m, ControlMonsterTypeActiveInit, false)
}

func StopControlMonsterBody(m monster.Model) packet.Encode {
	// Reset never reaches the post-mobId aggro byte; pass false for clarity.
	return ControlMonsterBody(m, ControlMonsterTypeReset, false)
}

func ControlMonsterBody(m monster.Model, controlType ControlMonsterType, aggro bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			var mem packetmodel.MonsterModel
			if controlType > ControlMonsterTypeReset {
				// Snap mob position before encoding so the v83 client doesn't
				// drop the mob through the floor on control assignment. See
				// data/map.SnapMobPosition for the rationale.
				x, y := dmap.SnapMobPosition(l, ctx, m.MapId(), m.X(), m.Y(), m.Fh())
				mem = packetmodel.NewMonster(x, y, m.Stance(), m.Fh(), packetmodel.MonsterAppearTypeRegen, m.Team())
				stat := buildMonsterTemporaryStat(l, t, m)
				mem.SetTemporaryStat(stat)
			}
			return monsterpkt.NewMonsterControl(monsterpkt.ControlType(controlType), m.UniqueId(), m.MonsterId(), mem, aggro).Encode(l, ctx)(options)
		}
	}
}
