package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MessengerOperationWriter = "MessengerOperation"

type JoinW struct {
	mode     byte
	position byte
}

func NewMessengerJoin(mode byte, position byte) JoinW {
	return JoinW{mode: mode, position: position}
}

func (m JoinW) Mode() byte     { return m.mode }
func (m JoinW) Position() byte { return m.position }

func (m JoinW) Operation() string { return MessengerOperationWriter }

func (m JoinW) String() string {
	return fmt.Sprintf("messenger join position [%d]", m.position)
}

func (m JoinW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		return w.Bytes()
	}
}

func (m *JoinW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
	}
}
