package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type TradeConfirmEntry struct {
	data uint32
	crc  uint32
}

func (e TradeConfirmEntry) Data() uint32 { return e.data }
func (e TradeConfirmEntry) Crc() uint32  { return e.crc }

type OperationTradeConfirm struct {
	entries []TradeConfirmEntry
}

func (m OperationTradeConfirm) Entries() []TradeConfirmEntry { return m.entries }

func (m OperationTradeConfirm) Operation() string { return "OperationTradeConfirm" }

func (m OperationTradeConfirm) String() string {
	return fmt.Sprintf("entries [%v]", m.entries)
}

func (m OperationTradeConfirm) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.entries)))
		for _, e := range m.entries {
			w.WriteInt(e.data)
			w.WriteInt(e.crc)
		}
		return w.Bytes()
	}
}

func (m *OperationTradeConfirm) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		size := r.ReadByte()
		m.entries = make([]TradeConfirmEntry, 0, size)
		for i := byte(0); i < size; i++ {
			data := r.ReadUint32()
			crc := r.ReadUint32()
			m.entries = append(m.entries, TradeConfirmEntry{data: data, crc: crc})
		}
	}
}
