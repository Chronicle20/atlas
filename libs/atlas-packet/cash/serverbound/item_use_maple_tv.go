package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseMapleTV is the USE_CASH_ITEM sub-body for Maple TV items
// (5075xxx / 5074000 on GMS>=95). tvType is derived by the CALLER from the
// item id (itemId % 10) — it is not on the wire; it selects which prefix
// fields exist. Cosmic-derived (UseCashItemHandler case 5); per-version IDA
// verification in task-123 phases 19-20.
type ItemUseMapleTV struct {
	tvType          byte
	pad             byte // present only when tvType == 3 (meaning unknown; drained)
	ear             bool
	receiverName    string
	lines           [5]string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMapleTV(updateTimeFirst bool, tvType byte) *ItemUseMapleTV {
	return &ItemUseMapleTV{updateTimeFirst: updateTimeFirst, tvType: tvType}
}

func (m ItemUseMapleTV) TvType() byte         { return m.tvType }
func (m ItemUseMapleTV) Ear() bool            { return m.ear }
func (m ItemUseMapleTV) ReceiverName() string { return m.receiverName }
func (m ItemUseMapleTV) Lines() []string      { return m.lines[:] }
func (m ItemUseMapleTV) UpdateTime() uint32   { return m.updateTime }

func (m ItemUseMapleTV) Operation() string { return "ItemUseMapleTV" }

func (m ItemUseMapleTV) String() string {
	return fmt.Sprintf("tvType [%d] ear [%t] receiverName [%s] lines %v updateTime [%d]",
		m.tvType, m.ear, m.receiverName, m.lines, m.updateTime)
}

func (m ItemUseMapleTV) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if m.tvType != 1 {
			if m.tvType >= 3 {
				if m.tvType == 3 {
					w.WriteByte(m.pad)
				}
				w.WriteBool(m.ear)
			} else if m.tvType != 2 {
				w.WriteByte(m.pad)
			}
			if m.tvType != 4 {
				w.WriteAsciiString(m.receiverName)
			}
		}
		for _, ln := range m.lines {
			w.WriteAsciiString(ln)
		}
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMapleTV) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		if m.tvType != 1 {
			if m.tvType >= 3 {
				if m.tvType == 3 {
					m.pad = r.ReadByte()
				}
				m.ear = r.ReadBool()
			} else if m.tvType != 2 {
				m.pad = r.ReadByte()
			}
			if m.tvType != 4 {
				m.receiverName = r.ReadAsciiString()
			}
		}
		for i := range m.lines {
			m.lines[i] = r.ReadAsciiString()
		}
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
