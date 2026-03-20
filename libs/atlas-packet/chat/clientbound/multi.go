package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MultiChatWriter = "CharacterChatMulti"

type MultiChat struct {
	mode    byte
	from    string
	message string
}

func NewMultiChat(mode byte, from string, message string) MultiChat {
	return MultiChat{mode: mode, from: from, message: message}
}

func (m MultiChat) Mode() byte      { return m.mode }
func (m MultiChat) From() string    { return m.from }
func (m MultiChat) Message() string { return m.message }

func (m MultiChat) Operation() string { return MultiChatWriter }
func (m MultiChat) String() string {
	return fmt.Sprintf("multi chat mode [%d] from [%s]", m.mode, m.from)
}

func (m MultiChat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.from)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *MultiChat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.from = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
