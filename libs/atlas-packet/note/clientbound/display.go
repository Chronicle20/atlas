package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/note"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Display - mode, notes
type Display struct {
	mode  byte
	notes []note.NoteEntry
}

func NewNoteDisplay(mode byte, notes []note.NoteEntry) Display {
	return Display{mode: mode, notes: notes}
}

func (m Display) Mode() byte          { return m.mode }
func (m Display) Notes() []note.NoteEntry  { return m.notes }
func (m Display) Operation() string   { return NoteOperationWriter }

func (m Display) String() string {
	return fmt.Sprintf("note display entries [%d]", len(m.notes))
}

func (m Display) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.notes)))
		for _, n := range m.notes {
			w.WriteByteArray(n.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *Display) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := int(r.ReadByte())
		m.notes = make([]note.NoteEntry, count)
		for i := 0; i < count; i++ {
			m.notes[i].Decode(l, ctx)(r, options)
		}
	}
}
