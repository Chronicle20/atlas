package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	fieldcb "github.com/Chronicle20/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func WarpToMapBody(channelId channel.Id, mapId _map.Id, portalId uint32, hp uint16) packet.Encode {
	return fieldcb.NewWarpToMap(channelId, mapId, byte(portalId), hp).Encode
}

func SetFieldBody(channelId channel.Id, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			cd := BuildCharacterData(c, bl)
			return fieldcb.NewSetField(channelId, cd).Encode(l, ctx)(options)
		}
	}
}

const (
	ZeroTime int64 = 94354848000000000
)
