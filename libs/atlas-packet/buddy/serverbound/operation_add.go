package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationAdd struct {
	name  string
	group string
}

func (m OperationAdd) Name() string {
	return m.name
}

func (m OperationAdd) Group() string {
	return m.group
}

func (m OperationAdd) Operation() string {
	return "OperationAdd"
}

func (m OperationAdd) String() string {
	return fmt.Sprintf("name [%s] group [%s]", m.name, m.group)
}

func (m OperationAdd) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteAsciiString(m.group)
		return w.Bytes()
	}
}

func (m *OperationAdd) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.group = r.ReadAsciiString()
	}
}
