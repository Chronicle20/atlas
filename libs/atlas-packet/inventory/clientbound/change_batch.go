package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/inventory"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ChangeBatch - multi-operation inventory change with structured entries.
type ChangeBatch struct {
	silent  bool
	entries []inventory.ChangeEntry
}

func NewChangeBatch(silent bool, entries ...inventory.ChangeEntry) ChangeBatch {
	return ChangeBatch{silent: silent, entries: entries}
}

func (m ChangeBatch) Silent() bool                    { return m.silent }
func (m ChangeBatch) Entries() []inventory.ChangeEntry { return m.entries }
func (m ChangeBatch) Operation() string               { return InventoryChangeWriter }

func (m ChangeBatch) String() string {
	return fmt.Sprintf("change batch entries [%d]", len(m.entries))
}

func (m ChangeBatch) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.silent)
		w.WriteByte(byte(len(m.entries)))
		addMov := int8(-1)
		for _, entry := range m.entries {
			w.WriteByteArray(entry.EncodeEntry(l, ctx)(options))
			if entry.EntryAddMov() > addMov {
				addMov = entry.EntryAddMov()
			}
		}
		if addMov > -1 {
			w.WriteInt8(addMov)
		}
		return w.Bytes()
	}
}

func (m *ChangeBatch) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.silent = !r.ReadBool()
		count := int(r.ReadByte())
		m.entries = make([]inventory.ChangeEntry, 0, count)
		for i := 0; i < count; i++ {
			entry := inventory.DecodeChangeEntry(l, ctx, r, options)
			if entry != nil {
				m.entries = append(m.entries, entry)
			}
		}
		// Read addMov if any entry indicates equipment state change
		for _, entry := range m.entries {
			if entry.EntryAddMov() > -1 {
				_ = r.ReadInt8()
				break
			}
		}
	}
}
