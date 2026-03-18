package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationFieldAddToBlackList struct {
	name string
}

func (m OperationFieldAddToBlackList) Name() string { return m.name }

func (m OperationFieldAddToBlackList) Operation() string { return "OperationFieldAddToBlackList" }

func (m OperationFieldAddToBlackList) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m OperationFieldAddToBlackList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *OperationFieldAddToBlackList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
