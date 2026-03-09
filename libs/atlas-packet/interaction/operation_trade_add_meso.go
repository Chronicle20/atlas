package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationTradeAddMeso struct {
	amount int32
}

func (m OperationTradeAddMeso) Amount() int32 { return m.amount }

func (m OperationTradeAddMeso) Operation() string { return "OperationTradeAddMeso" }

func (m OperationTradeAddMeso) String() string {
	return fmt.Sprintf("amount [%d]", m.amount)
}

func (m OperationTradeAddMeso) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.amount)
		return w.Bytes()
	}
}

func (m *OperationTradeAddMeso) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.amount = r.ReadInt32()
	}
}
