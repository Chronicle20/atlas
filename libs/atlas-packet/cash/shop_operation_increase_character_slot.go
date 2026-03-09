package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationIncreaseCharacterSlotHandle = "CashShopOperationIncreaseCharacterSlotHandle"

// ShopOperationIncreaseCharacterSlot - CCashShop::SendIncCharSlotCount
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

func (m ShopOperationIncreaseCharacterSlot) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.isPoints)
		w.WriteInt(m.currency)
		w.WriteInt(m.serialNumber)
		return w.Bytes()
	}
}

func (m *ShopOperationIncreaseCharacterSlot) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.isPoints = r.ReadBool()
		m.currency = r.ReadUint32()
		m.serialNumber = r.ReadUint32()
	}
}
