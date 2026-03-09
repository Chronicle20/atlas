package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationChat struct {
	message string
}

func (m OperationChat) Message() string { return m.message }

func (m OperationChat) Operation() string { return "OperationChat" }

func (m OperationChat) String() string {
	return fmt.Sprintf("message [%s]", m.message)
}

func (m OperationChat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *OperationChat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
	}
}
