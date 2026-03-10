package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerLoadWriter = "ServerLoad"

type ServerLoad struct {
	code byte
}

func NewServerLoad(code byte) ServerLoad {
	return ServerLoad{code: code}
}

func (m ServerLoad) Code() byte          { return m.code }
func (m ServerLoad) Operation() string   { return ServerLoadWriter }
func (m ServerLoad) String() string      { return fmt.Sprintf("code [%d]", m.code) }

func (m ServerLoad) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *ServerLoad) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
	}
}
