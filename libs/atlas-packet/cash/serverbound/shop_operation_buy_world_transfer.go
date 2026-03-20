package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationBuyWorldTransferHandle = "CashShopOperationBuyWorldTransferHandle"

// ShopOperationBuyWorldTransfer - CCashShop::SendBuyWorldTransfer
type ShopOperationBuyWorldTransfer struct {
	serialNumber uint32
	targetWorld  uint32
}

func (m ShopOperationBuyWorldTransfer) SerialNumber() uint32 { return m.serialNumber }
func (m ShopOperationBuyWorldTransfer) TargetWorld() uint32  { return m.targetWorld }

func (m ShopOperationBuyWorldTransfer) Operation() string {
	return CashShopOperationBuyWorldTransferHandle
}

func (m ShopOperationBuyWorldTransfer) String() string {
	return fmt.Sprintf("serialNumber [%d], targetWorld [%d]", m.serialNumber, m.targetWorld)
}

func (m ShopOperationBuyWorldTransfer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		w.WriteInt(m.targetWorld)
		return w.Bytes()
	}
}

func (m *ShopOperationBuyWorldTransfer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
		m.targetWorld = r.ReadUint32()
	}
}
