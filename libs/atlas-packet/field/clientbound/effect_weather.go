package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const FieldEffectWeatherWriter = "FieldEffectWeather"

type EffectWeather struct {
	active   bool
	itemId   uint32
	message  string
	extra    uint32
	hasExtra bool
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

func (m EffectWeather) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "JMS" {
			m.encodeJMS(w)
		} else {
			m.encodeGMS(w)
		}
		return w.Bytes()
	}
}

func (m EffectWeather) encodeGMS(w *response.Writer) {
	w.WriteBool(!m.active)
	w.WriteInt(m.itemId)
	if m.active {
		w.WriteAsciiString(m.message)
	}
}

func (m EffectWeather) encodeJMS(w *response.Writer) {
	w.WriteInt(m.itemId)
	if m.hasExtra {
		w.WriteInt(m.extra)
	}
	if m.itemId != 0 {
		w.WriteAsciiString(m.message)
	}
}

func (m *EffectWeather) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "JMS" {
			m.decodeJMS(r)
		} else {
			m.decodeGMS(r)
		}
	}
}

func (m *EffectWeather) decodeGMS(r *request.Reader) {
	m.active = !r.ReadBool()
	m.itemId = r.ReadUint32()
	if m.active {
		m.message = r.ReadAsciiString()
	}
}

func (m *EffectWeather) decodeJMS(r *request.Reader) {
	m.itemId = r.ReadUint32()
	if m.hasExtra {
		m.extra = r.ReadUint32()
	}
	if m.itemId != 0 {
		m.message = r.ReadAsciiString()
	}
}
