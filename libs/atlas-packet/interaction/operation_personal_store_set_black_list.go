package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationPersonalStoreSetBlackList struct {
	entries []byte
}

func (m OperationPersonalStoreSetBlackList) Entries() []byte { return m.entries }

func (m OperationPersonalStoreSetBlackList) Operation() string {
	return "OperationPersonalStoreSetBlackList"
}

func (m OperationPersonalStoreSetBlackList) String() string {
	return fmt.Sprintf("entries [%v]", m.entries)
}

func (m OperationPersonalStoreSetBlackList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(uint16(len(m.entries)))
		for _, e := range m.entries {
			w.WriteByte(e)
		}
		return w.Bytes()
	}
}

func (m *OperationPersonalStoreSetBlackList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		size := r.ReadUint16()
		m.entries = make([]byte, 0, size)
		for i := uint16(0); i < size; i++ {
			m.entries = append(m.entries, r.ReadByte())
		}
	}
}
