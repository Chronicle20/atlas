package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type TransactionEntry struct {
	data uint32
	crc  uint32
}

func (e TransactionEntry) Data() uint32 { return e.data }
func (e TransactionEntry) Crc() uint32  { return e.crc }

type OperationTransaction struct {
	entries []TransactionEntry
}

func (m OperationTransaction) Entries() []TransactionEntry { return m.entries }

func (m OperationTransaction) Operation() string { return "OperationTransaction" }

func (m OperationTransaction) String() string {
	return fmt.Sprintf("entries [%v]", m.entries)
}

func (m OperationTransaction) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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

func (m *OperationTransaction) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		size := r.ReadByte()
		m.entries = make([]TransactionEntry, 0, size)
		for i := byte(0); i < size; i++ {
			data := r.ReadUint32()
			crc := r.ReadUint32()
			m.entries = append(m.entries, TransactionEntry{data: data, crc: crc})
		}
	}
}
