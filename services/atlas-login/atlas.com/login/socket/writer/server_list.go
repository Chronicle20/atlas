package writer

import (
	"atlas-login/socket/model"
	"atlas-login/world"
	"fmt"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

const ServerListEntry = "ServerListEntry"
const ServerListEnd = "ServerListEnd"

func ServerListEntryBody(tenant tenant.Model) func(worldId byte, worldName string, state world.State, eventMessage string, channelLoad []model.Load) BodyProducer {
	return func(worldId byte, worldName string, state world.State, eventMessage string, channelLoad []model.Load) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(worldId)
			w.WriteAsciiString(worldName)

			if tenant.Region() == "GMS" {
				if tenant.MajorVersion() > 12 {
					w.WriteByte(byte(state))
					w.WriteAsciiString(eventMessage)
					w.WriteShort(100) // eventExpRate 100 = 1x
					w.WriteShort(100) // eventDropRate 100 = 1x

					//support blocking character creation
					w.WriteByte(0)
				}
			} else if tenant.Region() == "JMS" {
				w.WriteByte(byte(state))
				w.WriteAsciiString(eventMessage)
				w.WriteShort(100) // eventExpRate 100 = 1x
				w.WriteShort(100) // eventDropRate 100 = 1x
			}

			w.WriteByte(byte(len(channelLoad)))
			for _, x := range channelLoad {
				w.WriteAsciiString(fmt.Sprintf("%s - %d", worldName, x.ChannelId()))
				w.WriteInt(x.Capacity())
				w.WriteByte(1)
				w.WriteByte(x.ChannelId() - 1)
				w.WriteBool(false) // adult channel
			}

			//balloon size
			if tenant.Region() == "GMS" {
				if tenant.MajorVersion() > 12 {
					w.WriteShort(0)
				}
			} else if tenant.Region() == "JMS" {
				w.WriteShort(0)
			}

			// for loop
			// w.WriteShort // x
			// w.WriteShort // y
			// w.WriteAsciiString // message
			return w.Bytes()
		}
	}
}

func ServerListEndBody(w *response.Writer, _ map[string]interface{}) []byte {
	w.WriteByte(byte(0xFF))
	return w.Bytes()
}
