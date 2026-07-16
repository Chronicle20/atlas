package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperationIncreaseCharacterSlotHandle = "CashShopOperationIncreaseCharacterSlotHandle"

// ShopOperationIncreaseCharacterSlot - CCashShop::SendIncCharSlotCount
// packet-audit:fname CCashShop::OnIncCharacterSlotCount
type ShopOperationIncreaseCharacterSlot struct {
	isPoints     bool
	currency     uint32
	serialNumber uint32
}

func (m ShopOperationIncreaseCharacterSlot) IsPoints() bool       { return m.isPoints }
func (m ShopOperationIncreaseCharacterSlot) Currency() uint32      { return m.currency }
func (m ShopOperationIncreaseCharacterSlot) SerialNumber() uint32  { return m.serialNumber }

func (m ShopOperationIncreaseCharacterSlot) Operation() string {
	return CashShopOperationIncreaseCharacterSlotHandle
}

func (m ShopOperationIncreaseCharacterSlot) String() string {
	return fmt.Sprintf("isPoints [%t], currency [%d], serialNumber [%d]", m.isPoints, m.currency, m.serialNumber)
}

func (m ShopOperationIncreaseCharacterSlot) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.isPoints)
		// v79 CCashShop::OnIncCharacterSlotCount@0x4673be: COutPacket(221)
		// Encode1(9)=mode (routed op), Encode1(v30==2)=isPoints, Encode4(a2)=
		// serialNumber. No currency int (that was added at/after v83).
		if !legacyGMS(t) {
			w.WriteInt(m.currency)
		}
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationIncreaseCharacterSlot) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.isPoints = r.ReadBool()
		if !legacyGMS(t) {
			m.currency = r.ReadUint32()
		}
		m.serialNumber = r.ReadUint32()
	}
}
