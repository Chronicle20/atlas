package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldEffectWeather = "FieldEffectWeather"

func FieldEffectWeatherStartBody(itemId uint32, message string) packet.Encode {
	return FieldEffectWeatherBody(true, itemId, message)
}

func FieldEffectWeatherEndBody(itemId uint32) packet.Encode {
	return FieldEffectWeatherBody(false, itemId, "")
}

func FieldEffectWeatherBody(active bool, itemId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteBool(!active)
			w.WriteInt(itemId)
			if active {
				w.WriteAsciiString(message)
			}
			return w.Bytes()
		}
	}
}
