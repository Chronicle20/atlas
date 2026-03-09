package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationOpen struct {
	success bool
}

func (m OperationOpen) Success() bool { return m.success }

func (m OperationOpen) Operation() string { return "OperationOpen" }

func (m OperationOpen) String() string {
	return fmt.Sprintf("success [%v]", m.success)
}

func (m OperationOpen) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *OperationOpen) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.success = r.ReadBool()
	}
}
