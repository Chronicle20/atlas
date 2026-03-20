package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMerchantPutItem struct {
	inventoryType byte
	slot          int16
	quantity      uint16
	set           uint16
	price         uint32
}

func (m OperationMerchantPutItem) InventoryType() byte { return m.inventoryType }
func (m OperationMerchantPutItem) Slot() int16         { return m.slot }
func (m OperationMerchantPutItem) Quantity() uint16    { return m.quantity }
func (m OperationMerchantPutItem) Set() uint16         { return m.set }
func (m OperationMerchantPutItem) Price() uint32       { return m.price }

func (m OperationMerchantPutItem) Operation() string { return "OperationMerchantPutItem" }

func (m OperationMerchantPutItem) String() string {
	return fmt.Sprintf("inventoryType [%d] slot [%d] quantity [%d] set [%d] price [%d]", m.inventoryType, m.slot, m.quantity, m.set, m.price)
}

func (m OperationMerchantPutItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		w.WriteShort(m.quantity)
		w.WriteShort(m.set)
		w.WriteInt(m.price)
		return w.Bytes()
	}
}

func (m *OperationMerchantPutItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadInt16()
		m.quantity = r.ReadUint16()
		m.set = r.ReadUint16()
		m.price = r.ReadUint32()
	}
}
