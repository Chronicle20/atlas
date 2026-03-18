package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterNameResponseWriter = "CharacterNameResponse"

type CharacterNameResponse struct {
	name string
	code byte
}

func NewCharacterNameResponse(name string, code byte) CharacterNameResponse {
	return CharacterNameResponse{name: name, code: code}
}

func (m CharacterNameResponse) Name() string     { return m.name }
func (m CharacterNameResponse) Code() byte       { return m.code }
func (m CharacterNameResponse) Operation() string { return CharacterNameResponseWriter }
func (m CharacterNameResponse) String() string {
	return fmt.Sprintf("name [%s], code [%d]", m.name, m.code)
}

func (m CharacterNameResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *CharacterNameResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.code = r.ReadByte()
	}
}
