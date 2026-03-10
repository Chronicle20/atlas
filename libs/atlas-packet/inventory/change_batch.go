package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ChangeBatch - multi-operation inventory change with pre-encoded entries
type ChangeBatch struct {
	silent     bool
	entryBytes [][]byte
	addMov     int8
}

func NewChangeBatch(silent bool, entryBytes [][]byte, addMov int8) ChangeBatch {
	return ChangeBatch{silent: silent, entryBytes: entryBytes, addMov: addMov}
}

func (m ChangeBatch) Silent() bool        { return m.silent }
func (m ChangeBatch) EntryBytes() [][]byte { return m.entryBytes }
func (m ChangeBatch) AddMov() int8        { return m.addMov }
func (m ChangeBatch) Operation() string   { return InventoryChangeWriter }

func (m ChangeBatch) String() string {
	return fmt.Sprintf("change batch entries [%d]", len(m.entryBytes))
}

func (m ChangeBatch) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.silent)
		w.WriteByte(byte(len(m.entryBytes)))
		for _, entry := range m.entryBytes {
			w.WriteByteArray(entry)
		}
		if m.addMov > -1 {
			w.WriteInt8(m.addMov)
		}
		return w.Bytes()
	}
}

func (m *ChangeBatch) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
