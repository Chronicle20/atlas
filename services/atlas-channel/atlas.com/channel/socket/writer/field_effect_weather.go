package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldEffectWeather = "FieldEffectWeather"

func FieldEffectWeatherStartBody(l logrus.FieldLogger) func(itemId uint32, message string) BodyProducer {
	return func(itemId uint32, message string) BodyProducer {
		return FieldEffectWeatherBody(l)(true, itemId, message)
	}
}

func FieldEffectWeatherEndBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return FieldEffectWeatherBody(l)(false, 0, "")
	}
}

func FieldEffectWeatherBody(l logrus.FieldLogger) func(active bool, itemId uint32, message string) BodyProducer {
	return func(active bool, itemId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(active)
			w.WriteInt(itemId)
			if active {
				w.WriteAsciiString(message)
			}
			return w.Bytes()
		}
	}
}
