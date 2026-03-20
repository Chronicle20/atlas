package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ShopBuy struct {
	slot          uint16
	itemId        uint32
	quantity      uint16
	discountPrice uint32
}

func (m ShopBuy) Slot() uint16         { return m.slot }
func (m ShopBuy) ItemId() uint32       { return m.itemId }
func (m ShopBuy) Quantity() uint16     { return m.quantity }
func (m ShopBuy) DiscountPrice() uint32 { return m.discountPrice }

func (m ShopBuy) Operation() string { return "ShopBuy" }

func (m ShopBuy) String() string {
	return fmt.Sprintf("slot [%d] itemId [%d] quantity [%d] discountPrice [%d]", m.slot, m.itemId, m.quantity, m.discountPrice)
}

func (m ShopBuy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.slot)
		w.WriteInt(m.itemId)
		w.WriteShort(m.quantity)
		w.WriteInt(m.discountPrice)
		return w.Bytes()
	}
}

func (m *ShopBuy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadUint16()
		m.itemId = r.ReadUint32()
		m.quantity = r.ReadUint16()
		m.discountPrice = r.ReadUint32()
	}
}
