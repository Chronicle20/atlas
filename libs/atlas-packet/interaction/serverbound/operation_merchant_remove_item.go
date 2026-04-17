package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMerchantRemoveItem struct {
	index uint16
}

func (m OperationMerchantRemoveItem) Index() uint16 { return m.index }

func (m OperationMerchantRemoveItem) Operation() string { return "OperationMerchantRemoveItem" }

func (m OperationMerchantRemoveItem) String() string {
	return fmt.Sprintf("index [%d]", m.index)
}

func (m OperationMerchantRemoveItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.index)
		return w.Bytes()
	}
}

func (m *OperationMerchantRemoveItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.index = r.ReadUint16()
	}
}
