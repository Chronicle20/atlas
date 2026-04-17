package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Error struct {
	mode byte
	name string
}

func NewError(mode byte, name string) Error {
	return Error{mode: mode, name: name}
}

func (m Error) Mode() byte  { return m.mode }
func (m Error) Name() string { return m.name }

func (m Error) Operation() string {
	return PartyOperationWriter
}

func (m Error) String() string {
	return fmt.Sprintf("mode [%d], name [%s]", m.mode, m.name)
}

func (m Error) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *Error) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
