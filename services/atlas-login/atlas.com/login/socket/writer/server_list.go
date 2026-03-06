package writer

import (
	"atlas-login/socket/model"
	"atlas-login/world"
	"context"
	"fmt"

	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const ServerListEntry = "ServerListEntry"
const ServerListEnd = "ServerListEnd"

func ServerListEntryBody(worldId world2.Id, worldName string, state world.State, eventMessage string, channelLoad []model.Load) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(worldId))
			w.WriteAsciiString(worldName)

			if t.Region() == "GMS" {
				if t.MajorVersion() > 12 {
					w.WriteByte(byte(state))
					w.WriteAsciiString(eventMessage)
					w.WriteShort(100) // eventExpRate 100 = 1x
					w.WriteShort(100) // eventDropRate 100 = 1x

					//support blocking character creation
					w.WriteByte(0)
				}
			} else if t.Region() == "JMS" {
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
				w.WriteByte(byte(x.ChannelId() - 1))
				w.WriteBool(false) // adult channel
			}

			//balloon size
			if t.Region() == "GMS" {
				if t.MajorVersion() > 12 {
					w.WriteShort(0)
				}
			} else if t.Region() == "JMS" {
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

func ServerListEndBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(0xFF))
			return w.Bytes()
		}
	}
}
