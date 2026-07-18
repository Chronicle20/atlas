package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
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
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()))
			return fieldcb.NewSetField(channelId, cd).Encode(l, ctx)(options)
		}
	}
}

const (
	ZeroTime int64 = 94354848000000000
)
