package writer

import (
	"atlas-login/socket/model"
	"atlas-login/world"
	"context"

	world2 "github.com/Chronicle20/atlas-constants/world"
	loginpkt "github.com/Chronicle20/atlas-packet/login"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func ServerListEntryBody(worldId world2.Id, worldName string, state world.State, eventMessage string, channelLoad []model.Load) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			cls := make([]packetmodel.ChannelLoad, len(channelLoad))
			for i, x := range channelLoad {
				cls[i] = packetmodel.NewChannelLoad(x.ChannelId(), x.Capacity())
			}
			return loginpkt.NewServerListEntry(worldId, worldName, byte(state), eventMessage, cls).Encode(l, ctx)(options)
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
