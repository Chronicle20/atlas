package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Chat struct {
	mode    byte
	message string
}

func NewMessengerChat(mode byte, message string) Chat {
	return Chat{mode: mode, message: message}
}

func (m Chat) Mode() byte      { return m.mode }
func (m Chat) Message() string  { return m.message }

func (m Chat) Operation() string { return MessengerOperationWriter }

func (m Chat) String() string {
	return fmt.Sprintf("messenger chat message [%s]", m.message)
}

func (m Chat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *Chat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}
