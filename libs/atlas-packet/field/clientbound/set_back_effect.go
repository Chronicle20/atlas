package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SetBackEffectWriter = "SetBackEffect"

// Back-effect selectors. These are wire constants proven by the decompile
// (task-281 design §1.1): the client branches on nEffect and tweens the
// resolved back-layer page's alpha to 255 (show) or 0 (hide). Any other
// value makes CMapLoadable::OnSetBackEffect return without touching the
// field, which is why atlas-maps rejects it at the command consumer.
const (
	BackEffectShow byte = 0
	BackEffectHide byte = 1
)

// SetBackEffect is the clientbound CMapLoadable::OnSetBackEffect packet. The
// client reads four fields and tweens the alpha of every IWzGr2DLayer on
// back-layer page pageId to 255 (effect 0) or 0 (effect 1) over duration
// milliseconds. duration is a FADE length, not a lifetime. fieldId is decoded
// but unread by the v95 handler; it is kept because it occupies four wire
// bytes and omitting it would desynchronise the stream.
// packet-audit:fname CMapLoadable::OnSetBackEffect
type SetBackEffect struct {
	effect   byte
	fieldId  uint32
	pageId   byte
	duration uint32
}

func NewSetBackEffect(effect byte, fieldId uint32, pageId byte, duration uint32) SetBackEffect {
	return SetBackEffect{effect: effect, fieldId: fieldId, pageId: pageId, duration: duration}
}

func (m SetBackEffect) Effect() byte     { return m.effect }
func (m SetBackEffect) FieldId() uint32  { return m.fieldId }
func (m SetBackEffect) PageId() byte     { return m.pageId }
func (m SetBackEffect) Duration() uint32 { return m.duration }

func (m SetBackEffect) Operation() string { return SetBackEffectWriter }
func (m SetBackEffect) String() string {
	return fmt.Sprintf("effect [%d] fieldId [%d] pageId [%d] duration [%d]", m.effect, m.fieldId, m.pageId, m.duration)
}

func (m SetBackEffect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.effect)
		w.WriteInt(m.fieldId)
		w.WriteByte(m.pageId)
		w.WriteInt(m.duration)
		return w.Bytes()
	}
}

func (m *SetBackEffect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.effect = r.ReadByte()
		m.fieldId = r.ReadUint32()
		m.pageId = r.ReadByte()
		m.duration = r.ReadUint32()
	}
}
