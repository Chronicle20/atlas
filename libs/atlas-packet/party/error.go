package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ErrorW struct {
	mode byte
	name string
}

func NewErrorW(mode byte, name string) ErrorW {
	return ErrorW{mode: mode, name: name}
}

func (m ErrorW) Mode() byte  { return m.mode }
func (m ErrorW) Name() string { return m.name }

func (m ErrorW) Operation() string {
	return PartyOperationWriter
}

func (m ErrorW) String() string {
	return fmt.Sprintf("mode [%d], name [%s]", m.mode, m.name)
}

func (m ErrorW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *ErrorW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
