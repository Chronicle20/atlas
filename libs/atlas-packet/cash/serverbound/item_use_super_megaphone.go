package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseSuperMegaphone is the USE_CASH_ITEM sub-body for the Super Megaphone
// (5072xxx). Cosmic-derived (UseCashItemHandler case 2); per-version IDA
// verification in task-123 phases 19-20.
type ItemUseSuperMegaphone struct {
	message         string
	whisper         bool
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseSuperMegaphone(updateTimeFirst bool) *ItemUseSuperMegaphone {
	return &ItemUseSuperMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseSuperMegaphone) Message() string    { return m.message }
func (m ItemUseSuperMegaphone) Whisper() bool      { return m.whisper }
func (m ItemUseSuperMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseSuperMegaphone) Operation() string { return "ItemUseSuperMegaphone" }

func (m ItemUseSuperMegaphone) String() string {
	return fmt.Sprintf("message [%s] whisper [%t] updateTime [%d]", m.message, m.whisper, m.updateTime)
}

func (m ItemUseSuperMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		w.WriteBool(m.whisper)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseSuperMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		m.whisper = r.ReadBool()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
