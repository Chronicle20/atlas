package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ShopSell struct {
	slot     int16
	itemId   uint32
	quantity uint16
}

func (m ShopSell) Slot() int16      { return m.slot }
func (m ShopSell) ItemId() uint32   { return m.itemId }
func (m ShopSell) Quantity() uint16 { return m.quantity }

func (m ShopSell) Operation() string { return "ShopSell" }

func (m ShopSell) String() string {
	return fmt.Sprintf("slot [%d] itemId [%d] quantity [%d]", m.slot, m.itemId, m.quantity)
}

func (m ShopSell) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

func (m *ShopSell) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.quantity = r.ReadUint16()
	}
}
