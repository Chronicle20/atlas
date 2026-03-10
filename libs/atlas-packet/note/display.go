package note

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Display - mode, noteEntryBytes
type Display struct {
	mode           byte
	noteEntryBytes [][]byte
}

func NewNoteDisplay(mode byte, noteEntryBytes [][]byte) Display {
	return Display{mode: mode, noteEntryBytes: noteEntryBytes}
}

func (m Display) Mode() byte             { return m.mode }
func (m Display) NoteEntryBytes() [][]byte { return m.noteEntryBytes }
func (m Display) Operation() string       { return NoteOperationWriter }

func (m Display) String() string {
	return fmt.Sprintf("note display entries [%d]", len(m.noteEntryBytes))
}

func (m Display) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.noteEntryBytes)))
		for _, entry := range m.noteEntryBytes {
			w.WriteByteArray(entry)
		}
		return w.Bytes()
	}
}

func (m *Display) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
