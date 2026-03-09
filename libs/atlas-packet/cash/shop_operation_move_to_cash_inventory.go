package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationMoveToCashInventoryHandle = "CashShopOperationMoveToCashInventoryHandle"

// ShopOperationMoveToCashInventory - CCashShop::SendTransferToCashInventory
type ShopOperationMoveToCashInventory struct {
	serialNumber  uint64
	inventoryType byte
}

func (m ShopOperationMoveToCashInventory) SerialNumber() uint64  { return m.serialNumber }
func (m ShopOperationMoveToCashInventory) InventoryType() byte   { return m.inventoryType }

func (m ShopOperationMoveToCashInventory) Operation() string {
	return CashShopOperationMoveToCashInventoryHandle
}

func (m ShopOperationMoveToCashInventory) String() string {
	return fmt.Sprintf("serialNumber [%d], inventoryType [%d]", m.serialNumber, m.inventoryType)
}

func (m ShopOperationMoveToCashInventory) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.serialNumber)
		w.WriteByte(m.inventoryType)
		return w.Bytes()
	}
}

func (m *ShopOperationMoveToCashInventory) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint64()
		m.inventoryType = r.ReadByte()
	}
}
