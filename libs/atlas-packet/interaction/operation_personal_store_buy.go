package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationPersonalStoreBuy struct {
	index    byte
	quantity uint16
}

func (m OperationPersonalStoreBuy) Index() byte { return m.index }

func (m OperationPersonalStoreBuy) Quantity() uint16 { return m.quantity }

func (m OperationPersonalStoreBuy) Operation() string { return "OperationPersonalStoreBuy" }

func (m OperationPersonalStoreBuy) String() string {
	return fmt.Sprintf("index [%d], quantity [%d]", m.index, m.quantity)
}

func (m OperationPersonalStoreBuy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.index)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

func (m *OperationPersonalStoreBuy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.index = r.ReadByte()
		m.quantity = r.ReadUint16()
	}
}
