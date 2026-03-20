package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationFieldRemoveFromBlackList struct {
	name string
}

func (m OperationFieldRemoveFromBlackList) Name() string { return m.name }

func (m OperationFieldRemoveFromBlackList) Operation() string {
	return "OperationFieldRemoveFromBlackList"
}

func (m OperationFieldRemoveFromBlackList) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m OperationFieldRemoveFromBlackList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *OperationFieldRemoveFromBlackList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
