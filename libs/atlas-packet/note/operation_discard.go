package note

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DiscardEntry struct {
	id   uint32
	flag byte
}

func (e DiscardEntry) Id() uint32 {
	return e.id
}

func (e DiscardEntry) Flag() byte {
	return e.flag
}

type OperationDiscard struct {
	count   byte
	val1    byte
	val2    byte
	entries []DiscardEntry
}

func (m OperationDiscard) Count() byte {
	return m.count
}

func (m OperationDiscard) Val1() byte {
	return m.val1
}

func (m OperationDiscard) Val2() byte {
	return m.val2
}

func (m OperationDiscard) Entries() []DiscardEntry {
	return m.entries
}

func (m OperationDiscard) Operation() string {
	return "OperationDiscard"
}

func (m OperationDiscard) String() string {
	return fmt.Sprintf("count [%d] val1 [%d] val2 [%d] entries [%d]", m.count, m.val1, m.val2, len(m.entries))
}

func (m OperationDiscard) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.count)
		w.WriteByte(m.val1)
		w.WriteByte(m.val2)
		for _, e := range m.entries {
			w.WriteInt(e.id)
			w.WriteByte(e.flag)
		}
		return w.Bytes()
	}
}

func (m *OperationDiscard) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadByte()
		m.val1 = r.ReadByte()
		m.val2 = r.ReadByte()
		m.entries = make([]DiscardEntry, 0, m.count)
		for i := byte(0); i < m.count; i++ {
			e := DiscardEntry{
				id:   r.ReadUint32(),
				flag: r.ReadByte(),
			}
			m.entries = append(m.entries, e)
		}
	}
}
