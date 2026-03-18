package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMerchantBuy struct {
	index    byte
	quantity uint16
}

func (m OperationMerchantBuy) Index() byte       { return m.index }
func (m OperationMerchantBuy) Quantity() uint16   { return m.quantity }

func (m OperationMerchantBuy) Operation() string { return "OperationMerchantBuy" }

func (m OperationMerchantBuy) String() string {
	return fmt.Sprintf("index [%d] quantity [%d]", m.index, m.quantity)
}

func (m OperationMerchantBuy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.index)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

func (m *OperationMerchantBuy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.index = r.ReadByte()
		m.quantity = r.ReadUint16()
	}
}
