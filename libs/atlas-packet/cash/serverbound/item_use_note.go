package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseNote - the note (memo send) arm of USE_CASH_ITEM, consume cash item
// type 0x15 (classification 509). The client opens a send-memo dialog and
// encodes recipient + message; updateTime trails on GMS v83/v84 only.
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest#Note
type ItemUseNote struct {
	toName          string
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseNote(updateTimeFirst bool) *ItemUseNote {
	return &ItemUseNote{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseNote) ToName() string     { return m.toName }
func (m ItemUseNote) Message() string    { return m.message }
func (m ItemUseNote) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseNote) Operation() string { return "ItemUseNote" }

func (m ItemUseNote) String() string {
	return fmt.Sprintf("toName [%s] message [%s] updateTime [%d]", m.toName, m.message, m.updateTime)
}

func (m ItemUseNote) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.toName)
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseNote) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.toName = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
