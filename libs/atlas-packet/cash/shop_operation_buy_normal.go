package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyNormalHandle = "CashShopOperationBuyNormalHandle"

// ShopOperationBuyNormal - CCashShop::SendBuyNormal
type ShopOperationBuyNormal struct {
	serialNumber uint32
}

func (m ShopOperationBuyNormal) SerialNumber() uint32 { return m.serialNumber }

func (m ShopOperationBuyNormal) Operation() string {
	return CashShopOperationBuyNormalHandle
}

func (m ShopOperationBuyNormal) String() string {
	return fmt.Sprintf("serialNumber [%d]", m.serialNumber)
}

func (m ShopOperationBuyNormal) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyNormal) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
	}
}
