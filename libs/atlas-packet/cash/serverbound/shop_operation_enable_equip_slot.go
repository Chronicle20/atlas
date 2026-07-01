package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationEnableEquipSlotHandle = "CashShopOperationEnableEquipSlotHandle"

// ShopOperationEnableEquipSlot - CCashShop::SendEnableEquipSlotExt
// packet-audit:fname CCashShop::OnEnableEquipSlotExt
type ShopOperationEnableEquipSlot struct {
	pointType    bool
	currency     uint32 // v79 legacy currency bitmask int
	flag         byte   // v79 legacy constant byte (client sends 1)
	serialNumber uint32
}

func (m ShopOperationEnableEquipSlot) PointType() bool      { return m.pointType }
func (m ShopOperationEnableEquipSlot) Currency() uint32     { return m.currency }
func (m ShopOperationEnableEquipSlot) Flag() byte           { return m.flag }
func (m ShopOperationEnableEquipSlot) SerialNumber() uint32 { return m.serialNumber }

func (m ShopOperationEnableEquipSlot) Operation() string {
	return CashShopOperationEnableEquipSlotHandle
}

func (m ShopOperationEnableEquipSlot) String() string {
	return fmt.Sprintf("pointType [%t], serialNumber [%d]", m.pointType, m.serialNumber)
}

func (m ShopOperationEnableEquipSlot) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// v79 CCashShop::OnEnableEquipSlotExt@0x469fa9: COutPacket(221)
		// Encode1(mode 6|7) (routed op), Encode1(v45==2)=pointType/isPoints,
		// Encode4(v45)=currency, Encode1(1)=constant flag, Encode4(a2)=serialNumber.
		if legacyGMS(t) {
			w.WriteBool(m.pointType)
			w.WriteInt(m.currency)
			w.WriteByte(m.flag)
			w.WriteInt(m.serialNumber)
			return w.Bytes()
		}
		w.WriteBool(m.pointType)
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationEnableEquipSlot) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if legacyGMS(t) {
			m.pointType = r.ReadBool()
			m.currency = r.ReadUint32()
			m.flag = r.ReadByte()
			m.serialNumber = r.ReadUint32()
			return
		}
		m.pointType = r.ReadBool()
		m.serialNumber = r.ReadUint32()
	}
}
