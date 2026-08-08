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
		} else if t.MajorVersion() == 61 {
			m.encodeGMSLegacy(w)
		} else {
			m.encodeGMS(w)
		}
		return w.Bytes()
	}
}

// encodeGMSLegacy is the itemId-first GMS BLOW_WEATHER wire, with NO leading bool.
// IDA (gms_v61): sub_4ED39C reads Decode4(itemId) @0x4ed3ae, checks the item type
// (sub_48A4D6 == 19), then DecodeStr(message) @0x4ed40f — no leading byte at all.
// An active start populates itemId+message; an end sends itemId only.
//
// task-188 corrected the gate from `< 61` to `== 61`. The old gate sent this shape
// to v48, but v48's real handler — CField::OnBlowWeather = sub_4C930A @0x4c930a, the
// case-'U'(85) arm of CField::OnPacket @0x4c66f2 — reads Decode1 @0x4c9328 BEFORE
// Decode4(itemId) @0x4c932e, and gates DecodeStr(message) @0x4c9558 on that leading
// byte being 0. That is exactly encodeGMS's shape, so v48 now takes it.
//
// The prior comment cited sub_4C95F2 @0x4c95f2 as v48's OnBlowWeather; the IDB names
// that address ?OnPlayJukeBox@CField@@ and it is the case-'V'(86) arm — a different
// packet. The no-leading-bool claim was read off the jukebox handler.
//
// UNVERIFIED: v72 and v79 have not been checked and currently take encodeGMS. If
// they share v61's itemId-first shape this gate needs widening.
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
		} else if t.MajorVersion() == 61 {
			m.decodeGMSLegacy(r)
		} else {
			m.decodeGMS(r)
		}
	}
}

func (m *EffectWeather) decodeGMSLegacy(r *request.Reader) {
	m.itemId = r.ReadUint32()
	// This shape has no leading active/end bool; the v61 client gates the message
	// read on the item being a weather-type item (sub_48A4D6 == 19). An active start
	// carries a trailing message, an end carries none — mirror encodeGMSLegacy by
	// keying on remaining bytes so the round-trip is symmetric.
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
