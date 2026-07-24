package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const FieldEffectWeatherWriter = "FieldEffectWeather"

// packet-audit:fname CField::OnBlowWeather
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
		} else if t.MajorVersion() < 61 {
			m.encodeGMSLegacy(w)
		} else {
			m.encodeGMS(w)
		}
		return w.Bytes()
	}
}

// encodeGMSLegacy is the pre-v61 GMS BLOW_WEATHER wire. IDA: v48 CField::OnBlowWeather
// = sub_4C95F2 @0x4c95f2 reads Decode4(itemId) @0x4c9604 then — for a weather-type
// item (sub_47742E==18 && itemId>=0) — DecodeStr(message) @0x4c9669, with NO leading
// bool byte (the leading `!active` bool is a v83+ addition; v61's sub_4ED39C reads the
// same itemId-first shape). An active start populates itemId+message; an end sends
// itemId only. Gated < 61 so v61+ stay on encodeGMS unchanged.
func (m EffectWeather) encodeGMSLegacy(w *response.Writer) {
	w.WriteInt(m.itemId)
	if m.active {
		w.WriteAsciiString(m.message)
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
		} else if t.MajorVersion() < 61 {
			m.decodeGMSLegacy(r)
		} else {
			m.decodeGMS(r)
		}
	}
}

func (m *EffectWeather) decodeGMSLegacy(r *request.Reader) {
	m.itemId = r.ReadUint32()
	// Legacy has no leading active/end bool; the v48 client gates the message read
	// on the item being a weather-type item. An active start carries a trailing
	// message, an end carries none — mirror encodeGMSLegacy by keying on remaining
	// bytes so the round-trip is symmetric.
	m.active = r.Available() > 0
	if m.active {
		m.message = r.ReadAsciiString()
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
