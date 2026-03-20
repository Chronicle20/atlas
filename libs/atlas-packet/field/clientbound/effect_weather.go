package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldEffectWeatherWriter = "FieldEffectWeather"

type EffectWeather struct {
	active  bool
	itemId  uint32
	message string
}

func NewFieldEffectWeatherStart(itemId uint32, message string) EffectWeather {
	return EffectWeather{active: true, itemId: itemId, message: message}
}

func NewFieldEffectWeatherEnd(itemId uint32) EffectWeather {
	return EffectWeather{active: false, itemId: itemId}
}

func (m EffectWeather) Operation() string { return FieldEffectWeatherWriter }
func (m EffectWeather) String() string {
	return fmt.Sprintf("active [%t], itemId [%d]", m.active, m.itemId)
}

func (m EffectWeather) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.active)
		w.WriteInt(m.itemId)
		if m.active {
			w.WriteAsciiString(m.message)
		}
		return w.Bytes()
	}
}

func (m *EffectWeather) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.active = !r.ReadBool()
		m.itemId = r.ReadUint32()
		if m.active {
			m.message = r.ReadAsciiString()
		}
	}
}
