package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationAnswerInvite struct {
	messengerId uint32
}

func (m OperationAnswerInvite) MessengerId() uint32 {
	return m.messengerId
}

func (m OperationAnswerInvite) Operation() string {
	return "OperationAnswerInvite"
}

func (m OperationAnswerInvite) String() string {
	return fmt.Sprintf("messengerId [%d]", m.messengerId)
}

func (m OperationAnswerInvite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.messengerId)
		return w.Bytes()
	}
}

func (m *OperationAnswerInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.messengerId = r.ReadUint32()
	}
}
