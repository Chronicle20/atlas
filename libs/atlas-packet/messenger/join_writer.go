package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MessengerOperationWriter = "MessengerOperation"

type Join struct {
	mode     byte
	position byte
}

func NewMessengerJoin(mode byte, position byte) Join {
	return Join{mode: mode, position: position}
}

func (m Join) Mode() byte     { return m.mode }
func (m Join) Position() byte { return m.position }

func (m Join) Operation() string { return MessengerOperationWriter }

func (m Join) String() string {
	return fmt.Sprintf("messenger join position [%d]", m.position)
}

func (m Join) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.position)
		return w.Bytes()
	}
}

func (m *Join) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.position = r.ReadByte()
	}
}
