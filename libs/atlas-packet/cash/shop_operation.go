package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationHandle = "CashShopOperationHandle"

// ShopOperation - CCashShop
type ShopOperation struct {
	op byte
}

func (m ShopOperation) Op() byte { return m.op }

func (m ShopOperation) Operation() string {
	return CashShopOperationHandle
}

func (m ShopOperation) String() string {
	return fmt.Sprintf("op [%d]", m.op)
}

func (m ShopOperation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.op)
		return w.Bytes()
	}
}

func (m *ShopOperation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.op = r.ReadByte()
	}
}
