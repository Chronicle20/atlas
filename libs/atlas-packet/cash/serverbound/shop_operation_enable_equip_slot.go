package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationEnableEquipSlotHandle = "CashShopOperationEnableEquipSlotHandle"

// ShopOperationEnableEquipSlot - CCashShop::SendEnableEquipSlotExt
type ShopOperationEnableEquipSlot struct {
	pointType    bool
	serialNumber uint32
}

func (m ShopOperationEnableEquipSlot) PointType() bool       { return m.pointType }
func (m ShopOperationEnableEquipSlot) SerialNumber() uint32  { return m.serialNumber }

func (m ShopOperationEnableEquipSlot) Operation() string {
	return CashShopOperationEnableEquipSlotHandle
}

func (m ShopOperationEnableEquipSlot) String() string {
	return fmt.Sprintf("pointType [%t], serialNumber [%d]", m.pointType, m.serialNumber)
}

func (m ShopOperationEnableEquipSlot) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.pointType)
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationEnableEquipSlot) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.pointType = r.ReadBool()
		m.serialNumber = r.ReadUint32()
	}
}
