package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseMegaphone is the USE_CASH_ITEM sub-body for the basic Megaphone
// (5071xxx, cash-slot type 12 within classification 507).
// Cosmic-derived (UseCashItemHandler case 1); per-version IDA verification in task-123 phases 19-20.
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
type ItemUseMegaphone struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMegaphone(updateTimeFirst bool) *ItemUseMegaphone {
	return &ItemUseMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseMegaphone) Message() string    { return m.message }
func (m ItemUseMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseMegaphone) Operation() string { return "ItemUseMegaphone" }

func (m ItemUseMegaphone) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
