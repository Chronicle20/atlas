package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type RemoveW struct {
	mode     byte
	position byte
}

func NewMessengerRemove(mode byte, position byte) RemoveW {
	return RemoveW{mode: mode, position: position}
}

func (m RemoveW) Mode() byte     { return m.mode }
func (m RemoveW) Position() byte { return m.position }

func (m RemoveW) Operation() string { return MessengerOperationWriter }

func (m RemoveW) String() string {
	return fmt.Sprintf("messenger remove position [%d]", m.position)
}

func (m RemoveW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		return w.Bytes()
	}
}

func (m *RemoveW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
	}
}
