package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseAvatarMegaphone is the USE_CASH_ITEM sub-body for the Avatar
// Megaphone (item classification 539xxxx, cash-slot type 43 — IDA-verified
// via get_cashslot_item_type on gms_v95; the Cosmic-derived "5077xxx" guess
// in earlier revisions of this comment did not match: 5077xxx is Triple
// Megaphone, cash-slot type 61 on gms_v95). Encoded inline by
// SendConsumeCashItemUseRequest's jumptable case 43 (CUIAvatarMegaphone
// dialog): 4 lines then whisper, no other fields.
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
type ItemUseAvatarMegaphone struct {
	lines           [4]string
	whisper         bool
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseAvatarMegaphone(updateTimeFirst bool) *ItemUseAvatarMegaphone {
	return &ItemUseAvatarMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseAvatarMegaphone) Lines() []string    { return m.lines[:] }
func (m ItemUseAvatarMegaphone) Whisper() bool      { return m.whisper }
func (m ItemUseAvatarMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseAvatarMegaphone) Operation() string { return "ItemUseAvatarMegaphone" }

func (m ItemUseAvatarMegaphone) String() string {
	return fmt.Sprintf("lines %v whisper [%t] updateTime [%d]", m.lines, m.whisper, m.updateTime)
}

func (m ItemUseAvatarMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		for _, ln := range m.lines {
			w.WriteAsciiString(ln)
		}
		w.WriteBool(m.whisper)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseAvatarMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		for i := range m.lines {
			m.lines[i] = r.ReadAsciiString()
		}
		m.whisper = r.ReadBool()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
