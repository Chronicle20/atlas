package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationGetPurchaseRecordHandle = "CashShopOperationGetPurchaseRecordHandle"

// ShopOperationGetPurchaseRecord - CCashShop::SendGetPurchaseRecord
type ShopOperationGetPurchaseRecord struct {
	serialNumber uint32
}

func (m ShopOperationGetPurchaseRecord) SerialNumber() uint32 { return m.serialNumber }

func (m ShopOperationGetPurchaseRecord) Operation() string {
	return CashShopOperationGetPurchaseRecordHandle
}

func (m ShopOperationGetPurchaseRecord) String() string {
	return fmt.Sprintf("serialNumber [%d]", m.serialNumber)
}

func (m ShopOperationGetPurchaseRecord) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationGetPurchaseRecord) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
	}
}
