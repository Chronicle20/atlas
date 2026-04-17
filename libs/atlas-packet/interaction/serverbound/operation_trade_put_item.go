package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationTradePutItem struct {
	inventoryType byte
	slot          int16
	quantity      uint16
	targetSlot    byte
}

func (m OperationTradePutItem) InventoryType() byte { return m.inventoryType }

func (m OperationTradePutItem) Slot() int16 { return m.slot }

func (m OperationTradePutItem) Quantity() uint16 { return m.quantity }

func (m OperationTradePutItem) TargetSlot() byte { return m.targetSlot }

func (m OperationTradePutItem) Operation() string { return "OperationTradePutItem" }

func (m OperationTradePutItem) String() string {
	return fmt.Sprintf("inventoryType [%d], slot [%d], quantity [%d], targetSlot [%d]", m.inventoryType, m.slot, m.quantity, m.targetSlot)
}

func (m OperationTradePutItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		w.WriteShort(m.quantity)
		w.WriteByte(m.targetSlot)
		return w.Bytes()
	}
}

func (m *OperationTradePutItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadInt16()
		m.quantity = r.ReadUint16()
		m.targetSlot = r.ReadByte()
	}
}
