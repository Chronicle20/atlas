package writer

import (
	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
)

const FieldEffectWeather = "FieldEffectWeather"

func FieldEffectWeatherStartBody(itemId uint32, message string) packet.Encode {
	return fieldpkt.NewFieldEffectWeatherStart(itemId, message).Encode
}

func FieldEffectWeatherEndBody(itemId uint32) packet.Encode {
	return fieldpkt.NewFieldEffectWeatherEnd(itemId).Encode
}

func FieldEffectWeatherBody(active bool, itemId uint32, message string) packet.Encode {
	if active {
		return fieldpkt.NewFieldEffectWeatherStart(itemId, message).Encode
	}
	return fieldpkt.NewFieldEffectWeatherEnd(itemId).Encode
}
