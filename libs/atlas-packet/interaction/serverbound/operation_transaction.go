package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type TransactionEntry struct {
	data uint32
	crc  uint32
}

func (e TransactionEntry) Data() uint32 { return e.data }
func (e TransactionEntry) Crc() uint32  { return e.crc }

// packet-audit:fname CCashTradingRoomDlg::Trade
type OperationTransaction struct {
	entries []TransactionEntry
}

func (m OperationTransaction) Entries() []TransactionEntry { return m.entries }

func (m OperationTransaction) Operation() string { return "OperationTransaction" }

func (m OperationTransaction) String() string {
	return fmt.Sprintf("entries [%v]", m.entries)
}

func (m OperationTransaction) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if tradeCrcPresent(t) {
			w.WriteByte(byte(len(m.entries)))
			for _, e := range m.entries {
				w.WriteInt(e.data)
				w.WriteInt(e.crc)
			}
		}
		return w.Bytes()
	}
}

func (m *OperationTransaction) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if !tradeCrcPresent(t) {
			return
		}
		size := r.ReadByte()
		m.entries = make([]TransactionEntry, 0, size)
		for i := byte(0); i < size; i++ {
			data := r.ReadUint32()
			crc := r.ReadUint32()
			m.entries = append(m.entries, TransactionEntry{data: data, crc: crc})
		}
	}
}
