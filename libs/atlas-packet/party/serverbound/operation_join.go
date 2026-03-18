package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationJoin struct {
	partyId uint32
}

func (m OperationJoin) PartyId() uint32 {
	return m.partyId
}

func (m OperationJoin) Operation() string {
	return "OperationJoin"
}

func (m OperationJoin) String() string {
	return fmt.Sprintf("partyId [%d]", m.partyId)
}

func (m OperationJoin) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.partyId)
		return w.Bytes()
	}
}

func (m *OperationJoin) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.partyId = r.ReadUint32()
	}
}
