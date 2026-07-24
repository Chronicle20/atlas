package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/teleportrock"
	"atlas-channel/maps/location"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func WarpToMapBody(channelId channel.Id, mapId _map.Id, portalId uint32, hp uint16) packet.Encode {
	return fieldcb.NewWarpToMap(channelId, mapId, byte(portalId), hp).Encode
}

// WarpToPositionBody warps the character to an exact (x, y) coordinate in the
// target map via the SET_FIELD chase mechanism — used to land a Mystic Door
// user on the linked door's exact position rather than a named portal.
func WarpToPositionBody(channelId channel.Id, mapId _map.Id, hp uint16, x int16, y int16) packet.Encode {
	return fieldcb.NewWarpToPosition(channelId, mapId, hp, x, y).Encode
}

func SetFieldBody(channelId channel.Id, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			trm, err := teleportrock.NewProcessor(l, ctx).GetByCharacterId(c.Id())
			if err != nil {
				// Fail-open: a missing list must never block login (design §4.4).
				l.WithError(err).Warnf("Unable to fetch teleport-rock maps for character [%d]; sending empty lists.", c.Id())
				trm = teleportrock.Model{}
			}
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()), trm)
			return fieldcb.NewSetField(channelId, cd).Encode(l, ctx)(options)
		}
	}
}

const (
	ZeroTime int64 = 94354848000000000
)
