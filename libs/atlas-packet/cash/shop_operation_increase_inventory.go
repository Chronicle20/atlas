package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationIncreaseInventoryHandle = "CashShopOperationIncreaseInventoryHandle"

// ShopOperationIncreaseInventory - CCashShop::SendIncSlotCount
type ShopOperationIncreaseInventory struct {
	isPoints      bool
	currency      uint32
	item          bool
	inventoryType byte
	serialNumber  uint32
}

func (m ShopOperationIncreaseInventory) IsPoints() bool       { return m.isPoints }
func (m ShopOperationIncreaseInventory) Currency() uint32      { return m.currency }
func (m ShopOperationIncreaseInventory) Item() bool            { return m.item }
func (m ShopOperationIncreaseInventory) InventoryType() byte   { return m.inventoryType }
func (m ShopOperationIncreaseInventory) SerialNumber() uint32  { return m.serialNumber }

func (m ShopOperationIncreaseInventory) Operation() string {
	return CashShopOperationIncreaseInventoryHandle
}

func (m ShopOperationIncreaseInventory) String() string {
	return fmt.Sprintf("isPoints [%t], currency [%d], item [%t], inventoryType [%d], serialNumber [%d]", m.isPoints, m.currency, m.item, m.inventoryType, m.serialNumber)
}

func (m ShopOperationIncreaseInventory) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.isPoints)
		w.WriteInt(m.currency)
		w.WriteBool(m.item)
		if m.item {
			w.WriteInt(m.serialNumber)
		} else {
			w.WriteByte(m.inventoryType)
		}
		return w.Bytes()
	}
}

func (m *ShopOperationIncreaseInventory) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.isPoints = r.ReadBool()
		m.currency = r.ReadUint32()
		m.item = r.ReadBool()
		if m.item {
			m.serialNumber = r.ReadUint32()
		} else {
			m.inventoryType = r.ReadByte()
		}
	}
}
