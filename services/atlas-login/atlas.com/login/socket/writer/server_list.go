package writer

import (
	"atlas-login/socket/model"
	"atlas-login/world"
	"context"
	"sort"

	"github.com/sirupsen/logrus"

	world2 "github.com/Chronicle20/atlas/libs/atlas-constants/world"
	loginpkt "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ServerListEntryBody encodes the WORLD_INFORMATION channel-list body. The v83
// client never reads a channel number off the wire: it indexes its channel-name
// array by the channel entry's packet position (CWvsContext::SetWorldInfo
// @0xa02dde, fed from CLogin::SendLoginPacket @0x5f6d6a). So packet position
// must equal channel id — sort a copy of channelLoad ascending by ChannelId
// before encoding, regardless of the order the caller supplied it in.
func ServerListEntryBody(worldId world2.Id, worldName string, state world.State, eventMessage string, channelLoad []model.Load) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			ordered := make([]model.Load, len(channelLoad))
			copy(ordered, channelLoad)
			sort.Slice(ordered, func(i, j int) bool {
				return ordered[i].ChannelId() < ordered[j].ChannelId()
			})

			cls := make([]packetmodel.ChannelLoad, len(ordered))
			for i, x := range ordered {
				cls[i] = packetmodel.NewChannelLoad(x.ChannelId(), x.Capacity())
			}
			return loginpkt.NewServerListEntry(worldId, worldName, byte(state), eventMessage, cls, nil).Encode(l, ctx)(options)
		}
	}
}

func ServerListEndBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return loginpkt.ServerListEnd{}.Encode(l, ctx)(options)
		}
	}
}
