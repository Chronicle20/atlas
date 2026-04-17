package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationMoveFromCashInventoryHandle = "CashShopOperationMoveFromCashInventoryHandle"

// ShopOperationMoveFromCashInventory - CCashShop::SendTransferFromCashInventory
type ShopOperationMoveFromCashInventory struct {
	serialNumber  uint64
	inventoryType byte
	slot          int16
}

func (m ShopOperationMoveFromCashInventory) SerialNumber() uint64   { return m.serialNumber }
func (m ShopOperationMoveFromCashInventory) InventoryType() byte    { return m.inventoryType }
func (m ShopOperationMoveFromCashInventory) Slot() int16            { return m.slot }

func (m ShopOperationMoveFromCashInventory) Operation() string {
	return CashShopOperationMoveFromCashInventoryHandle
}

func (m ShopOperationMoveFromCashInventory) String() string {
	return fmt.Sprintf("serialNumber [%d], inventoryType [%d], slot [%d]", m.serialNumber, m.inventoryType, m.slot)
}

func (m ShopOperationMoveFromCashInventory) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.serialNumber)
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		return w.Bytes()
	}
}

func (m *ShopOperationMoveFromCashInventory) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint64()
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadInt16()
	}
}
