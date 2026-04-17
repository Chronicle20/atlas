package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PartyOperationHandle = "PartyOperationHandle"

type Operation struct {
	op byte
}

func (m Operation) Op() byte {
	return m.op
}

func (m Operation) Operation() string {
	return PartyOperationHandle
}

func (m Operation) String() string {
	return fmt.Sprintf("op [%d]", m.op)
}

func (m Operation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.op)
		return w.Bytes()
	}
}

func (m *Operation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.op = r.ReadByte()
	}
}
