package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationSend struct {
	toName  string
	message string
}

func (m OperationSend) ToName() string {
	return m.toName
}

func (m OperationSend) Message() string {
	return m.message
}

func (m OperationSend) Operation() string {
	return "OperationSend"
}

func (m OperationSend) String() string {
	return fmt.Sprintf("toName [%s] message [%s]", m.toName, m.message)
}

func (m OperationSend) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.toName)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *OperationSend) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.toName = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
