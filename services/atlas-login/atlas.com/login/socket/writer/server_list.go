package writer

import (
	"atlas-login/channel"
	"atlas-login/tenant"
	"atlas-login/world"
	"fmt"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerListEntry = "ServerListEntry"
const ServerListEnd = "ServerListEnd"

func ServerListEntryBody(l logrus.FieldLogger, tenant tenant.Model) func(worldId byte, worldName string, state world.State, eventMessage string, channelLoad []channel.Load) BodyProducer {
	return func(worldId byte, worldName string, state world.State, eventMessage string, channelLoad []channel.Load) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(worldId)
			w.WriteAsciiString(worldName)
			w.WriteByte(byte(state))
			w.WriteAsciiString(eventMessage)
			w.WriteShort(100) // eventExpRate 100 = 1x
			w.WriteShort(100) // eventDropRate 100 = 1x

			if tenant.Region == "GMS" {
				//support blocking character creation
				w.WriteByte(0)
			}

			w.WriteByte(byte(len(channelLoad)))
			for _, x := range channelLoad {
				w.WriteAsciiString(fmt.Sprintf("%s - %d", worldName, x.ChannelId()))
				w.WriteInt(uint32(x.Capacity()))
				w.WriteByte(1)
				w.WriteByte(x.ChannelId() - 1)
				w.WriteBool(false) // adult channel
			}

			//balloon size
			w.WriteShort(0)
			// for loop
			// w.WriteShort // x
			// w.WriteShort // y
			// w.WriteAsciiString // message
			return w.Bytes()
		}
	}
}

func ServerListEndBody(l logrus.FieldLogger) BodyProducer {
	return func(op uint16, options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteShort(op)
		w.WriteByte(byte(0xFF))
		return w.Bytes()
	}
}
